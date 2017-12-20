package concepts

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/husobee/vestigo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/twinj/uuid"
)

func TestSearchByIDsNoResults(t *testing.T) {
	serverMock := new(mockConceptSearchAPI)
	requestedUUIDs := []string{uuid.NewV4().String()}
	serverMock.On("getRequest").Return("tid_TestSearchByIDsNoResults", requestedUUIDs)
	serverMock.On("getResponse").Return(`{}`, http.StatusOK)

	server := serverMock.startServer(t)
	defer server.Close()

	search := NewSearch(&http.Client{}, server.URL+"/concepts")
	concepts, err := search.ByIDs("tid_TestSearchByIDsNoResults", requestedUUIDs...)

	assert.NoError(t, err)
	assert.Len(t, concepts, 0)
	serverMock.AssertExpectations(t) // failure here means the concordances API has not been called
}

func TestSearchByIDs(t *testing.T) {
	serverMock := new(mockConceptSearchAPI)
	requestedUUIDs := []string{uuid.NewV4().String()}
	serverMock.On("getRequest").Return("tid_TestSearchByIDs", requestedUUIDs)

	searchResp, err := ioutil.ReadFile("./_fixtures/search_response.json")
	require.NoError(t, err)
	serverMock.On("getResponse").Return(string(searchResp), http.StatusOK)

	server := serverMock.startServer(t)
	defer server.Close()

	search := NewSearch(&http.Client{}, server.URL+"/concepts")
	concepts, err := search.ByIDs("tid_TestSearchByIDs", requestedUUIDs...)

	assert.NoError(t, err)
	assert.Len(t, concepts, 1)
	serverMock.AssertExpectations(t) // failure here means the concordances API has not been called
}

func TestSearchNoIDsProvided(t *testing.T) {
	search := NewSearch(&http.Client{}, "/concepts")
	_, err := search.ByIDs("tid_TestSearchNoIDsProvided")

	assert.EqualError(t, err, ErrNoConceptsToSearch.Error())
}

func TestSearchAllIDsProvidedEmpty(t *testing.T) {
	search := NewSearch(&http.Client{}, "/concepts")
	_, err := search.ByIDs("tid_TestSearchNoIDsProvided", "", "", "", "")

	assert.EqualError(t, err, ErrConceptUUIDsAreEmpty.Error())
}

func TestSearchRequestURLInvalid(t *testing.T) {
	search := NewSearch(&http.Client{}, ":#")
	_, err := search.ByIDs("tid_TestSearchRequestURLInvalid", uuid.NewV4().String())

	assert.Error(t, err)
}

func TestSearchRequestFails(t *testing.T) {
	search := NewSearch(&http.Client{}, "#:")
	_, err := search.ByIDs("tid_TestSearchRequestFails", uuid.NewV4().String())

	assert.Error(t, err)
}

func TestSearchResponseFailed(t *testing.T) {
	serverMock := new(mockConceptSearchAPI)
	requestedUUIDs := []string{uuid.NewV4().String()}
	serverMock.On("getRequest").Return("tid_TestSearchResponseFailed", requestedUUIDs)
	serverMock.On("getResponse").Return(`{"message":"forbidden!!!!!"}`, http.StatusForbidden)

	server := serverMock.startServer(t)
	defer server.Close()

	search := NewSearch(&http.Client{}, server.URL+"/concepts")
	_, err := search.ByIDs("tid_TestSearchResponseFailed", requestedUUIDs...)

	assert.EqualError(t, err, "403 Forbidden: forbidden!!!!!")
	serverMock.AssertExpectations(t) // failure here means the concordances API has not been called
}

func TestSearchResponseInvalidJSON(t *testing.T) {
	serverMock := new(mockConceptSearchAPI)
	requestedUUIDs := []string{uuid.NewV4().String()}
	serverMock.On("getRequest").Return("tid_TestSearchResponseInvalidJSON", requestedUUIDs)
	serverMock.On("getResponse").Return(`{`, http.StatusOK)

	server := serverMock.startServer(t)
	defer server.Close()

	search := NewSearch(&http.Client{}, server.URL+"/concepts")
	_, err := search.ByIDs("tid_TestSearchResponseInvalidJSON", requestedUUIDs...)

	assert.Error(t, err)
	serverMock.AssertExpectations(t) // failure here means the concordances API has not been called
}

type mockConceptSearchAPI struct {
	mock.Mock
}

func (m *mockConceptSearchAPI) getRequest() (string, []string) {
	args := m.Called()
	return args.String(0), args.Get(1).([]string)
}

func (m *mockConceptSearchAPI) getResponse() (string, int) {
	args := m.Called()
	return args.String(0), args.Int(1)
}

func (m *mockConceptSearchAPI) startServer(t *testing.T) *httptest.Server {
	r := vestigo.NewRouter()
	r.Get("/concepts", func(w http.ResponseWriter, r *http.Request) {
		tid, expectedIDs := m.getRequest()

		assert.Equal(t, tid, r.Header.Get("X-Request-Id"))
		assert.Equal(t, expectedUserAgent, r.Header.Get("User-Agent"))

		query := r.URL.Query()
		actualIDs, found := query[conceptSearchQueryParam]
		assert.True(t, found)
		assert.Equal(t, expectedIDs, actualIDs)

		json, status := m.getResponse()
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write([]byte(json))
	})

	return httptest.NewServer(r)
}