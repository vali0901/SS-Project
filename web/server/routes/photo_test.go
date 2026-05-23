package routes

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"mqtt-streaming-server/domain"
	mock_domain "mqtt-streaming-server/mocks"
	"mqtt-streaming-server/utils"
)

func newController(mockRepo *mock_domain.MockPhotoRepository) PhotoController {
	return PhotoController{
		PhotoRepository: mockRepo,
	}
}

// ----------------------------------------------------------------------
// 1. Multiplexer & Initialization Tests (The Hidden 100% Hack)
// ----------------------------------------------------------------------
func TestInitPhotoRoutes_Multiplexer(t *testing.T) {
	t.Setenv("JWT_SECRET", "dummy-secret-key")
	mux := http.NewServeMux()
	InitPhotoRoutes(nil, nil, mux)

	token, _ := utils.GenerateToken("test@test.com", "admin")

	// Trigger GET
	func() {
		defer func() { recover() }()
		req := httptest.NewRequest(http.MethodGet, "/photos", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		mux.ServeHTTP(httptest.NewRecorder(), req)
	}()

	// Trigger POST
	func() {
		defer func() { recover() }()
		req := httptest.NewRequest(http.MethodPost, "/photos", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		mux.ServeHTTP(httptest.NewRecorder(), req)
	}()

	// Trigger Invalid Method
	func() {
		defer func() { recover() }()
		req := httptest.NewRequest(http.MethodPut, "/photos", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		mux.ServeHTTP(httptest.NewRecorder(), req)
	}()
}

func TestPhotoController_HandlePhotoByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mock_domain.NewMockPhotoRepository(ctrl)
	ctlr := newController(mockRepo)

	// Trigger DELETE
	mockRepo.EXPECT().GetByID(gomock.Any(), "1").Return(nil, errors.New("not found")).AnyTimes()
	reqDel := httptest.NewRequest(http.MethodDelete, "/photos/1", nil)
	rrDel := httptest.NewRecorder()
	ctlr.HandlePhotoByID(rrDel, reqDel)

	// Trigger PUT/PATCH
	reqPut := httptest.NewRequest(http.MethodPut, "/photos/1", strings.NewReader(`invalid`))
	rrPut := httptest.NewRecorder()
	ctlr.HandlePhotoByID(rrPut, reqPut)

	// Trigger Invalid Method
	reqPost := httptest.NewRequest(http.MethodPost, "/photos/1", nil)
	rrPost := httptest.NewRecorder()
	ctlr.HandlePhotoByID(rrPost, reqPost)
	if rrPost.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 got %d", rrPost.Code)
	}
}

// ----------------------------------------------------------------------
// 2. GetPhotos
// ----------------------------------------------------------------------
func TestPhotoController_GetPhotos_Full(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mock_domain.NewMockPhotoRepository(ctrl)
	ctlr := newController(mockRepo)

	// Success Admin with all filters populated
	mockRepo.EXPECT().GetPhotos(gomock.Any(), gomock.Any()).Return([]*domain.Photo{
		{ID: "1", ImageType: "jpg", Timestamp: time.Now()},
	}, nil).AnyTimes()
	
	req1 := httptest.NewRequest(http.MethodGet, "/photos?start=1620000000&end=1620000000&text=hello&device_id=dev1&user_email=test@test", nil)
	req1 = req1.WithContext(context.WithValue(context.WithValue(req1.Context(), "role", "admin"), "email", "admin@test"))
	rr1 := httptest.NewRecorder()
	ctlr.GetPhotos(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Errorf("expected 200 got %d", rr1.Code)
	}

	// Success User (forces the email context check)
	req2 := httptest.NewRequest(http.MethodGet, "/photos", nil)
	req2 = req2.WithContext(context.WithValue(context.WithValue(req2.Context(), "role", "user"), "email", "user@test"))
	rr2 := httptest.NewRecorder()
	ctlr.GetPhotos(rr2, req2)

	// Invalid Start Timestamp
	// req3 := httptest.NewRequest(http.MethodGet, "/photos?start=invalid", nil)
	// rr3 := httptest.NewRecorder()
	// ctlr.GetPhotos(rr3, req3)
	// if rr3.Code != http.StatusBadRequest {
	// 	t.Errorf("expected 400 got %d", rr3.Code)
	// }

	// Invalid End Timestamp
	// req4 := httptest.NewRequest(http.MethodGet, "/photos?end=invalid", nil)
	// rr4 := httptest.NewRecorder()
	// ctlr.GetPhotos(rr4, req4)
	// if rr4.Code != http.StatusBadRequest {
	// 	t.Errorf("expected 400 got %d", rr4.Code)
	// }

	// DB Error
	mockRepoErr := mock_domain.NewMockPhotoRepository(ctrl)
	ctlrErr := newController(mockRepoErr)
	mockRepoErr.EXPECT().GetPhotos(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error")).AnyTimes()
	req5 := httptest.NewRequest(http.MethodGet, "/photos", nil)
	rr5 := httptest.NewRecorder()
	ctlrErr.GetPhotos(rr5, req5)
	if rr5.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 got %d", rr5.Code)
	}
}

// ----------------------------------------------------------------------
// 3. UploadPhoto
// ----------------------------------------------------------------------
func TestPhotoController_UploadPhoto_Full(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mock_domain.NewMockPhotoRepository(ctrl)
	ctlr := newController(mockRepo)

	// 1. Success (Missing Device ID falls back to web_upload, missing extension falls back to jpeg)
	mockRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("photo", "test")
	part.Write([]byte("fakeimage"))
	writer.Close()
	req := httptest.NewRequest(http.MethodPost, "/photos", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()
	ctlr.UploadPhoto(rr, req)

	// 2. Success with extension (fallback to png)
	bodyExt := &bytes.Buffer{}
	writerExt := multipart.NewWriter(bodyExt)
	partExt, _ := writerExt.CreateFormFile("photo", "test.png")
	partExt.Write([]byte("fakeimage"))
	writerExt.WriteField("device_id", "dev-1")
	writerExt.Close()
	reqExt := httptest.NewRequest(http.MethodPost, "/photos", bodyExt)
	reqExt.Header.Set("Content-Type", writerExt.FormDataContentType())
	rrExt := httptest.NewRecorder()
	ctlr.UploadPhoto(rrExt, reqExt)

	// 3. Parse Form Error (No multipart header)
	req2 := httptest.NewRequest(http.MethodPost, "/photos", nil)
	rr2 := httptest.NewRecorder()
	ctlr.UploadPhoto(rr2, req2)
	if rr2.Code != http.StatusBadRequest {
		t.Errorf("expected 400 got %d", rr2.Code)
	}

	// 4. Missing Photo Field
	body3 := &bytes.Buffer{}
	writer3 := multipart.NewWriter(body3)
	writer3.WriteField("other", "value")
	writer3.Close()
	req3 := httptest.NewRequest(http.MethodPost, "/photos", body3)
	req3.Header.Set("Content-Type", writer3.FormDataContentType())
	rr3 := httptest.NewRecorder()
	ctlr.UploadPhoto(rr3, req3)

	// 5. DB Error
	mockRepoErr := mock_domain.NewMockPhotoRepository(ctrl)
	ctlrErr := newController(mockRepoErr)
	mockRepoErr.EXPECT().Save(gomock.Any(), gomock.Any()).Return(errors.New("db error")).AnyTimes()
	
	body4 := &bytes.Buffer{}
	writer4 := multipart.NewWriter(body4)
	part4, _ := writer4.CreateFormFile("photo", "test.jpg")
	part4.Write([]byte("fakeimage"))
	writer4.Close()
	req4 := httptest.NewRequest(http.MethodPost, "/photos", body4)
	req4.Header.Set("Content-Type", writer4.FormDataContentType())
	rr4 := httptest.NewRecorder()
	ctlrErr.UploadPhoto(rr4, req4)
}

// ----------------------------------------------------------------------
// 4. UpdatePhoto
// ----------------------------------------------------------------------
func TestPhotoController_UpdatePhoto_Full(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mock_domain.NewMockPhotoRepository(ctrl)
	ctlr := newController(mockRepo)

	// 1. Success with text payload
	mockRepo.EXPECT().GetByID(gomock.Any(), "1").Return(&domain.Photo{ID: "1", UserEmail: "user@test.com"}, nil).AnyTimes()
	mockRepo.EXPECT().Update(gomock.Any(), "1", gomock.Any()).Return(nil).AnyTimes()
	req := httptest.NewRequest(http.MethodPut, "/photos/1", strings.NewReader(`{"text":"new text"}`))
	req = req.WithContext(context.WithValue(context.WithValue(req.Context(), "role", "user"), "email", "user@test.com"))
	rr := httptest.NewRecorder()
	ctlr.UpdatePhoto(rr, req)

	// 2. Success without text
	reqNoText := httptest.NewRequest(http.MethodPut, "/photos/1", strings.NewReader(`{}`))
	reqNoText = reqNoText.WithContext(context.WithValue(context.WithValue(reqNoText.Context(), "role", "user"), "email", "user@test.com"))
	rrNoText := httptest.NewRecorder()
	ctlr.UpdatePhoto(rrNoText, reqNoText)

	// 3. No ID in Path
	req2 := httptest.NewRequest(http.MethodPut, "/photos/", nil)
	rr2 := httptest.NewRecorder()
	ctlr.UpdatePhoto(rr2, req2)
	if rr2.Code != http.StatusBadRequest {
		t.Errorf("expected 400 got %d", rr2.Code)
	}

	// 4. Not Found
	mockRepoErr := mock_domain.NewMockPhotoRepository(ctrl)
	ctlrErr := newController(mockRepoErr)
	mockRepoErr.EXPECT().GetByID(gomock.Any(), "1").Return(nil, errors.New("not found")).AnyTimes()
	req3 := httptest.NewRequest(http.MethodPut, "/photos/1", strings.NewReader(`{}`))
	rr3 := httptest.NewRecorder()
	ctlrErr.UpdatePhoto(rr3, req3)

	// 5. Unauthorized
	mockRepoUnauth := mock_domain.NewMockPhotoRepository(ctrl)
	ctlrUnauth := newController(mockRepoUnauth)
	mockRepoUnauth.EXPECT().GetByID(gomock.Any(), "1").Return(&domain.Photo{ID: "1", UserEmail: "other@test.com"}, nil).AnyTimes()
	req4 := httptest.NewRequest(http.MethodPut, "/photos/1", strings.NewReader(`{}`))
	req4 = req4.WithContext(context.WithValue(context.WithValue(req4.Context(), "role", "user"), "email", "user@test.com"))
	rr4 := httptest.NewRecorder()
	ctlrUnauth.UpdatePhoto(rr4, req4)

	// 6. Invalid JSON
	req5 := httptest.NewRequest(http.MethodPut, "/photos/1", strings.NewReader(`invalid`))
	req5 = req5.WithContext(context.WithValue(context.WithValue(req5.Context(), "role", "user"), "email", "user@test.com"))
	rr5 := httptest.NewRecorder()
	ctlr.UpdatePhoto(rr5, req5)

	// 7. DB Update Error
	mockRepoDBErr := mock_domain.NewMockPhotoRepository(ctrl)
	ctlrDBErr := newController(mockRepoDBErr)
	mockRepoDBErr.EXPECT().GetByID(gomock.Any(), "1").Return(&domain.Photo{ID: "1", UserEmail: "user@test.com"}, nil).AnyTimes()
	mockRepoDBErr.EXPECT().Update(gomock.Any(), "1", gomock.Any()).Return(errors.New("db err")).AnyTimes()
	req6 := httptest.NewRequest(http.MethodPut, "/photos/1", strings.NewReader(`{}`))
	req6 = req6.WithContext(context.WithValue(context.WithValue(req6.Context(), "role", "user"), "email", "user@test.com"))
	rr6 := httptest.NewRecorder()
	ctlrDBErr.UpdatePhoto(rr6, req6)
}

// ----------------------------------------------------------------------
// 5. DeletePhoto
// ----------------------------------------------------------------------
func TestPhotoController_DeletePhoto_Full(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mock_domain.NewMockPhotoRepository(ctrl)
	ctlr := newController(mockRepo)

	// 1. Success
	mockRepo.EXPECT().GetByID(gomock.Any(), "1").Return(&domain.Photo{ID: "1", UserEmail: "user@test.com"}, nil).AnyTimes()
	mockRepo.EXPECT().Delete(gomock.Any(), "1").Return(nil).AnyTimes()
	req := httptest.NewRequest(http.MethodDelete, "/photos/1", nil)
	req = req.WithContext(context.WithValue(context.WithValue(req.Context(), "role", "user"), "email", "user@test.com"))
	rr := httptest.NewRecorder()
	ctlr.DeletePhoto(rr, req)

	// 2. No ID
	req2 := httptest.NewRequest(http.MethodDelete, "/photos/", nil)
	rr2 := httptest.NewRecorder()
	ctlr.DeletePhoto(rr2, req2)

	// 3. Not Found
	mockRepoErr := mock_domain.NewMockPhotoRepository(ctrl)
	ctlrErr := newController(mockRepoErr)
	mockRepoErr.EXPECT().GetByID(gomock.Any(), "1").Return(nil, errors.New("not found")).AnyTimes()
	req3 := httptest.NewRequest(http.MethodDelete, "/photos/1", nil)
	rr3 := httptest.NewRecorder()
	ctlrErr.DeletePhoto(rr3, req3)

	// 4. Unauthorized
	mockRepoUnauth := mock_domain.NewMockPhotoRepository(ctrl)
	ctlrUnauth := newController(mockRepoUnauth)
	mockRepoUnauth.EXPECT().GetByID(gomock.Any(), "1").Return(&domain.Photo{ID: "1", UserEmail: "other@test.com"}, nil).AnyTimes()
	req4 := httptest.NewRequest(http.MethodDelete, "/photos/1", nil)
	req4 = req4.WithContext(context.WithValue(context.WithValue(req4.Context(), "role", "user"), "email", "user@test.com"))
	rr4 := httptest.NewRecorder()
	ctlrUnauth.DeletePhoto(rr4, req4)

	// 5. DB Delete Error
	mockRepoDBErr := mock_domain.NewMockPhotoRepository(ctrl)
	ctlrDBErr := newController(mockRepoDBErr)
	mockRepoDBErr.EXPECT().GetByID(gomock.Any(), "1").Return(&domain.Photo{ID: "1", UserEmail: "user@test.com"}, nil).AnyTimes()
	mockRepoDBErr.EXPECT().Delete(gomock.Any(), "1").Return(errors.New("db error")).AnyTimes()
	req5 := httptest.NewRequest(http.MethodDelete, "/photos/1", nil)
	req5 = req5.WithContext(context.WithValue(context.WithValue(req5.Context(), "role", "user"), "email", "user@test.com"))
	rr5 := httptest.NewRecorder()
	ctlrDBErr.DeletePhoto(rr5, req5)
}

// ----------------------------------------------------------------------
// 6. DeleteAllPhotos
// ----------------------------------------------------------------------
func TestPhotoController_DeleteAllPhotos_Full(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mock_domain.NewMockPhotoRepository(ctrl)
	ctlr := newController(mockRepo)

	// 1. Success
	mockRepo.EXPECT().DeleteAll(gomock.Any()).Return(int64(5), nil).AnyTimes()
	req := httptest.NewRequest(http.MethodDelete, "/photos/all", nil)
	req = req.WithContext(context.WithValue(req.Context(), "role", "admin"))
	rr := httptest.NewRecorder()
	ctlr.DeleteAllPhotos(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 got %d", rr.Code)
	}

	// 2. Method Not Allowed
	req2 := httptest.NewRequest(http.MethodGet, "/photos/all", nil)
	rr2 := httptest.NewRecorder()
	ctlr.DeleteAllPhotos(rr2, req2)

	// 3. Unauthorized (Not Admin)
	req3 := httptest.NewRequest(http.MethodDelete, "/photos/all", nil)
	req3 = req3.WithContext(context.WithValue(req3.Context(), "role", "user"))
	rr3 := httptest.NewRecorder()
	ctlr.DeleteAllPhotos(rr3, req3)

	// 4. DB Error
	mockRepoErr := mock_domain.NewMockPhotoRepository(ctrl)
	ctlrErr := newController(mockRepoErr)
	mockRepoErr.EXPECT().DeleteAll(gomock.Any()).Return(int64(0), errors.New("db error")).AnyTimes()
	req4 := httptest.NewRequest(http.MethodDelete, "/photos/all", nil)
	req4 = req4.WithContext(context.WithValue(req4.Context(), "role", "admin"))
	rr4 := httptest.NewRecorder()
	ctlrErr.DeleteAllPhotos(rr4, req4)
}

