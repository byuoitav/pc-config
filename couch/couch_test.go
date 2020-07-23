package couch

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	pcconfig "github.com/byuoitav/pc-config"
	"github.com/go-kivik/couchdb/v3"
	"github.com/go-kivik/kivik/v3"
	"github.com/go-kivik/kivikmock/v3"
	"github.com/google/go-cmp/cmp"
)

var mockCamera = pcconfig.Camera{
	DisplayName: "mock cam",
	TiltUp:      "https://tiltUp",
	TiltDown:    "https://tiltDown",
	PanLeft:     "https://panLeft",
	PanRight:    "https://panRight",
	PanTiltStop: "https://panTiltStop",
	ZoomIn:      "https://zoomIn",
	ZoomOut:     "https://zoomOut",
	ZoomStop:    "https://zoomStop",
	Stream:      "https://stream",
	Presets: []pcconfig.CameraPreset{
		{
			DisplayName: "mock preset 1",
			SetPreset:   "https://mock preset 1",
		},
		{
			DisplayName: "mock preset 2",
			SetPreset:   "https://mock preset 2",
		},
	},
}

func TestNew(t *testing.T) {
	_, err := New(context.Background(), "example")
	if err != nil {
		t.Fatalf(err.Error())
	}
}

func TestNewError(t *testing.T) {
	var urlErr *url.Error

	_, err := New(context.Background(), "bad host")
	if !errors.As(err, &urlErr) {
		t.Fatalf("expected a url.Error")
	}

	if urlErr.Op != "parse" {
		t.Fatalf("expected a parse error, got %s", urlErr.Op)
	}
}

func TestNewWithClientNoAuth(t *testing.T) {
	client, mock := kivikmock.NewT(t)

	mock.ExpectPing().WillReturn(true)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cs, err := NewWithClient(ctx, client)
	if err != nil {
		t.Fatalf("unable to create config service: %s", err)
	}

	up, err := cs.(*configService).client.Ping(ctx)
	if err != nil {
		t.Fatalf("got error pinging client: %s", err)
	}

	if !up {
		t.Fatalf("expected ping to return true, got false")
	}
}

// TODO test auth error
func TestNewClientWithAuth(t *testing.T) {
	client, mock := kivikmock.NewT(t)

	// TODO this doesn't work the way i think it does lol
	mock.ExpectAuthenticate().WithAuthenticator(couchdb.BasicAuth("uname", "pass"))
	mock.ExpectPing().WillReturn(true)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cs, err := NewWithClient(ctx, client, WithBasicAuth("uname", "pass"))
	if err != nil {
		t.Fatalf("unable to create config service: %s", err)
	}

	up, err := cs.(*configService).client.Ping(ctx)
	if err != nil {
		t.Fatalf("got error pinging client: %s", err)
	}

	if !up {
		t.Fatalf("expected ping to return true, got false")
	}
}

func TestNewClientWithAuthError(t *testing.T) {
	client, mock := kivikmock.NewT(t)

	expected := errors.New("auth error!")
	mock.ExpectAuthenticate().WillReturnError(expected)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := NewWithClient(ctx, client, WithBasicAuth("uname", "pass"))
	if !errors.Is(err, expected) {
		t.Fatalf("expected %q, got %q", expected, err)
	}
}

func TestRoomAndControlGroup(t *testing.T) {
	client, mock := kivikmock.NewT(t)

	db := mock.NewDB()
	mock.ExpectDB().WithName(_defaultPCMappingDB).WillReturn(db)
	db.ExpectGet().WithDocID("TEC-ITB-1101").WillReturn(kivikmock.DocumentT(t, `{
		"_id": "TEC-ITB-1101",
		"uiConfig": "ITB-1101",
		"controlGroup": "Test Control Group"
	}`))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cs, err := NewWithClient(ctx, client)
	if err != nil {
		t.Fatalf("unable to create config service: %s", err)
	}

	room, cg, err := cs.RoomAndControlGroup(ctx, "TEC-ITB-1101")
	if err != nil {
		t.Fatalf("failed to get room and control group: %s", err)
	}

	switch {
	case room != "ITB-1101":
		t.Fatalf("got wrong room: expected %q, got %q", "ITB-1101", room)
	case cg != "Test Control Group":
		t.Fatalf("got wrong control group: expected %q, got %q", "Test Control Group", cg)
	}
}

func TestRoomAndControlGroupError(t *testing.T) {
	client, mock := kivikmock.NewT(t)

	expected := errors.New("a bad error!")

	db := mock.NewDB()
	mock.ExpectDB().WithName(_defaultPCMappingDB).WillReturn(db)
	db.ExpectGet().WithDocID("TEC-ITB-1101").WillReturnError(expected)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cs, err := NewWithClient(ctx, client)
	if err != nil {
		t.Fatalf("unable to create config service: %s", err)
	}

	_, _, err = cs.RoomAndControlGroup(ctx, "TEC-ITB-1101")
	if !errors.Is(err, expected) {
		t.Fatalf("expected %q, got %q", expected, err)
	}
}

func TestRoomAndControlGroupRetry(t *testing.T) {
	client, mock := kivikmock.NewT(t)

	kerr := &kivik.Error{
		HTTPStatus: http.StatusNotFound,
		Err:        errors.New("error"),
	}

	db := mock.NewDB()
	mock.ExpectDB().WithName(_defaultPCMappingDB).WillReturn(db)
	db.ExpectGet().WithDocID("TEC-ITB-1101-NEW").WillReturnError(kerr)
	db.ExpectGet().WithDocID("TEC-ITB-1101-NE").WillReturnError(kerr)
	db.ExpectGet().WithDocID("TEC-ITB-1101-N").WillReturnError(kerr)
	db.ExpectGet().WithDocID("TEC-ITB-1101-").WillReturnError(kerr)
	db.ExpectGet().WithDocID("TEC-ITB-1101").WillReturn(kivikmock.DocumentT(t, `{
		"_id": "TEC-ITB-1101",
		"uiConfig": "ITB-1101",
		"controlGroup": "Test Control Group"
	}`))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cs, err := NewWithClient(ctx, client)
	if err != nil {
		t.Fatalf("unable to create config service: %s", err)
	}

	room, cg, err := cs.RoomAndControlGroup(ctx, "TEC-ITB-1101-NEW")
	if err != nil {
		t.Fatalf("failed to get room and control group: %s", err)
	}

	switch {
	case room != "ITB-1101":
		t.Fatalf("got wrong room: expected %q, got %q", "ITB-1101", room)
	case cg != "Test Control Group":
		t.Fatalf("got wrong control group: expected %q, got %q", "Test Control Group", cg)
	}
}

func TestRoomAndControlGroupRetryFail(t *testing.T) {
	client, mock := kivikmock.NewT(t)

	kerr := &kivik.Error{
		HTTPStatus: http.StatusNotFound,
		Err:        errors.New("error"),
	}
	expected := errors.New("some error")

	db := mock.NewDB()
	mock.ExpectDB().WithName(_defaultPCMappingDB).WillReturn(db)
	db.ExpectGet().WithDocID("TEC-ITB-1101-NEW").WillReturnError(kerr)
	db.ExpectGet().WithDocID("TEC-ITB-1101-NE").WillReturnError(kerr)
	db.ExpectGet().WithDocID("TEC-ITB-1101-N").WillReturnError(expected)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cs, err := NewWithClient(ctx, client)
	if err != nil {
		t.Fatalf("unable to create config service: %s", err)
	}

	_, _, err = cs.RoomAndControlGroup(ctx, "TEC-ITB-1101-NEW")
	if !errors.Is(err, expected) {
		t.Fatalf("expected %q, got %q", expected, err)
	}
}

func TestRoomAndControlGroupTooShort(t *testing.T) {
	client, mock := kivikmock.NewT(t)

	kerr := &kivik.Error{
		HTTPStatus: http.StatusNotFound,
		Err:        errors.New("error"),
	}

	db := mock.NewDB()
	mock.ExpectDB().WithName(_defaultPCMappingDB).WillReturn(db)
	db.ExpectGet().WithDocID("TEC-ITB-1101").WillReturnError(kerr)
	db.ExpectGet().WithDocID("TEC-ITB-110").WillReturnError(kerr)
	db.ExpectGet().WithDocID("TEC-ITB-11").WillReturnError(kerr)
	db.ExpectGet().WithDocID("TEC-ITB-1").WillReturnError(kerr)
	db.ExpectGet().WithDocID("TEC-ITB-").WillReturnError(kerr)
	db.ExpectGet().WithDocID("TEC-ITB").WillReturnError(kerr)
	db.ExpectGet().WithDocID("TEC-IT").WillReturnError(kerr)
	db.ExpectGet().WithDocID("TEC-I").WillReturnError(kerr)
	db.ExpectGet().WithDocID("TEC-").WillReturnError(kerr)
	db.ExpectGet().WithDocID("TEC").WillReturnError(kerr)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cs, err := NewWithClient(ctx, client)
	if err != nil {
		t.Fatalf("unable to create config service: %s", err)
	}

	_, _, err = cs.RoomAndControlGroup(ctx, "TEC-ITB-1101")
	if kivik.StatusCode(err) != http.StatusNotFound {
		t.Fatalf("got unexpected error: %s", err)
	}
}

func TestCameras(t *testing.T) {
	client, mock := kivikmock.NewT(t)

	db := mock.NewDB()
	mock.ExpectDB().WithName(_defaultUIConfigDB).WillReturn(db)
	db.ExpectGet().WithDocID("ITB-1101").WillReturn(kivikmock.DocumentT(t, uiConfig{
		ControlGroups: []struct {
			ID      string            `json:"name"`
			Cameras []pcconfig.Camera `json:"cameras"`
		}{
			{
				ID:      "Camera",
				Cameras: []pcconfig.Camera{mockCamera},
			},
		},
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cs, err := NewWithClient(ctx, client)
	if err != nil {
		t.Fatalf("unable to create config service: %s", err)
	}

	cameras, err := cs.Cameras(ctx, "ITB-1101", "Camera")
	if err != nil {
		t.Fatalf("failed to get room and control group: %s", err)
	}

	if len(cameras) != 1 {
		t.Fatalf("expected 1 camera, got %v", len(cameras))
	}

	if diff := cmp.Diff(mockCamera, cameras[0]); diff != "" {
		t.Errorf("generated incorrect mapping (-want, +got):\n%s", diff)
	}
}

func TestCameraInvalidConfig(t *testing.T) {
	client, mock := kivikmock.NewT(t)

	db := mock.NewDB()
	mock.ExpectDB().WithName(_defaultUIConfigDB).WillReturn(db)
	db.ExpectGet().WithDocID("ITB-1101").WillReturn(kivikmock.DocumentT(t, `{
		"presets": {
			"name": "a name",
			"cameras": [
				{
					"displayName": "good name",
					"presets": [
						{
							"displayName": false
						}
					]
				}
			]
		}
	}`))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cs, err := NewWithClient(ctx, client)
	if err != nil {
		t.Fatalf("unable to create config service: %s", err)
	}

	_, err = cs.Cameras(ctx, "ITB-1101", "Invalid Preset")
	if !strings.Contains(err.Error(), "unable to get/scan ui config") {
		t.Fatalf("expected unable to get/scan ui config error, got: %s", err.Error())
	}
}

func TestCameraMissing(t *testing.T) {
	client, mock := kivikmock.NewT(t)

	db := mock.NewDB()
	mock.ExpectDB().WithName(_defaultUIConfigDB).WillReturn(db)
	db.ExpectGet().WithDocID("ITB-1101").WillReturn(kivikmock.DocumentT(t, uiConfig{
		ControlGroups: []struct {
			ID      string            `json:"name"`
			Cameras []pcconfig.Camera `json:"cameras"`
		}{
			{
				ID:      "Camera",
				Cameras: []pcconfig.Camera{mockCamera},
			},
		},
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cs, err := NewWithClient(ctx, client)
	if err != nil {
		t.Fatalf("unable to create config service: %s", err)
	}

	_, err = cs.Cameras(ctx, "ITB-1101", "Invalid Preset")
	if !strings.Contains(err.Error(), "no matching control group found") {
		t.Fatalf("expected no matching control group error, got: %s", err.Error())
	}
}
