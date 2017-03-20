package service

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"code.uber.internal/infra/kraken/kraken/tracker/storage"

	bencode "github.com/jackpal/bencode-go"

	"code.uber.internal/infra/kraken/config/tracker"
	"code.uber.internal/infra/kraken/test/mocks/mock_storage"
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
)

type testMocks struct {
	appCfg    config.AppConfig
	ctrl      *gomock.Controller
	datastore *mock_storage.MockStorage
}

// mockController sets up all mocks and returns a teardown func that can be called with defer
func (m *testMocks) mockController(t gomock.TestReporter) func() {
	m.appCfg = config.AppConfig{}
	m.ctrl = gomock.NewController(t)
	m.datastore = mock_storage.NewMockStorage(m.ctrl)
	return func() {
		m.ctrl.Finish()
	}
}

func (m *testMocks) CreateHandler() http.Handler {
	return InitializeAPI(
		m.appCfg,
		m.datastore,
	)
}

func (m *testMocks) CreateHandlerAndServeRequest(request *http.Request) *http.Response {
	w := httptest.NewRecorder()
	m.CreateHandler().ServeHTTP(w, request)
	return w.Result()
}

func performRequest(handler http.Handler, request *http.Request) *http.Response {
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, request)
	return w.Result()
}

func TestAnnounceEndPoint(t *testing.T) {
	infoHash := "12345678901234567890"
	peerID := "ABCDEFGHIJKLMNOPQRST"
	port := "6881"
	ip := "255.255.255.255"
	downloaded := "1234"
	uploaded := "5678"
	left := "910"
	event := "stopped"

	bytesUploaded, _ := strconv.ParseInt(uploaded, 10, 64)
	bytesDownloaded, _ := strconv.ParseInt(downloaded, 10, 64)
	bytesLeft, _ := strconv.ParseInt(left, 10, 64)

	t.Run("Return 500 if missing parameters", func(t *testing.T) {
		announceRequest, _ := http.NewRequest("GET", "/announce", nil)

		mocks := &testMocks{}
		defer mocks.mockController(t)()

		response := mocks.CreateHandlerAndServeRequest(announceRequest)
		assert.Equal(t, 500, response.StatusCode)
	})
	t.Run("Return 200 and empty bencoded response", func(t *testing.T) {

		announceRequest, _ := http.NewRequest("GET",
			"/announce?info_hash="+infoHash+
				"&peer_id="+peerID+
				"&ip="+ip+
				"&port="+port+
				"&downloaded="+downloaded+
				"&uploaded="+uploaded+
				"&left="+left+
				"&event="+event, nil)

		mocks := &testMocks{}
		defer mocks.mockController(t)()

		mocks.datastore.EXPECT().Read(infoHash).Return([]storage.PeerInfo{}, nil)
		mocks.datastore.EXPECT().Update(
			&storage.PeerInfo{
				InfoHash:        infoHash,
				PeerID:          peerID,
				IP:              ip,
				Port:            port,
				BytesUploaded:   bytesUploaded,
				BytesDownloaded: bytesDownloaded,
				BytesLeft:       bytesLeft,
				Event:           event,
				Flags:           0}).Return(nil)
		response := mocks.CreateHandlerAndServeRequest(announceRequest)
		announceResponse := AnnouncerResponse{}
		bencode.Unmarshal(response.Body, &announceResponse)
		assert.Equal(t, announceResponse.Interval, int64(0))
		assert.Equal(t, announceResponse.Peers, []storage.PeerInfo{})
		assert.Equal(t, 200, response.StatusCode)
	})
	t.Run("Return 200 and single peer bencoded response", func(t *testing.T) {

		announceRequest, _ := http.NewRequest("GET",
			"/announce?info_hash="+infoHash+
				"&peer_id="+peerID+
				"&ip="+ip+
				"&port="+port+
				"&downloaded="+downloaded+
				"&uploaded="+uploaded+
				"&left="+left+
				"&event="+event, nil)

		mocks := &testMocks{}
		defer mocks.mockController(t)()

		peerFrom := storage.PeerInfo{
			InfoHash:        infoHash,
			PeerID:          peerID,
			IP:              ip,
			Port:            port,
			BytesUploaded:   bytesUploaded,
			BytesDownloaded: bytesDownloaded,
			BytesLeft:       bytesLeft,
			Event:           event,
			Flags:           0}

		peerTo := storage.PeerInfo{
			PeerID: peerID,
			IP:     ip,
			Port:   port}

		mocks.datastore.EXPECT().Read(infoHash).Return([]storage.PeerInfo{peerFrom}, nil)
		mocks.datastore.EXPECT().Update(&peerFrom).Return(nil)
		response := mocks.CreateHandlerAndServeRequest(announceRequest)
		announceResponse := AnnouncerResponse{}
		bencode.Unmarshal(response.Body, &announceResponse)
		assert.Equal(t, announceResponse.Interval, int64(0))
		assert.Equal(t, announceResponse.Peers, []storage.PeerInfo{peerTo})
		assert.Equal(t, 200, response.StatusCode)
	})

}
