// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/nanoninja/assert/require"
	"github.com/nanoninja/dojo/internal/httputil"
	"github.com/nanoninja/dojo/internal/middleware"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/service"
	"github.com/nanoninja/dojo/internal/store"
)

// ============================================================================
// Helpers
// ============================================================================

// testJWTSecret is the signing key used for JWT tokens in handler tests.
const testJWTSecret = "test-handler-jwt-secret-key-32b"

// Test UUIDs used across handler tests.
// Using UUIDv7-shaped values (time-ordered prefix) for realism.
const (
	testUserID      = "01966b0a-1234-7abc-def0-1234567890ab"
	testOtherUserID = "01966b0a-5678-7abc-def0-1234567890cd"
	testAdminID     = "01966b0a-9012-7abc-def0-1234567890ef"
	testUser1ID     = "01966b0a-1111-7abc-def0-1234567890aa"
	testUser2ID     = "01966b0a-2222-7abc-def0-1234567890bb"
	testUser3ID     = "01966b0a-3333-7abc-def0-1234567890cc"
	testNewUserID   = "01966b0a-4444-7abc-def0-1234567890dd"
)

// serve wraps h with httputil.Handle (discarding logs) and writes to w.
// This mirrors the production handler wiring without cluttering test output.
func serve(h httputil.HandlerFunc, w http.ResponseWriter, r *http.Request) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	httputil.Handle(h, logger).ServeHTTP(w, r)
}

// newJSONRequest creates an HTTP request with a JSON-encoded body.
func newJSONRequest(method, path string, body any) *http.Request {
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(method, path, bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	return r
}

// withChiParam injects a chi URL parameter into the request context.
func withChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// withUserID signs a JWT and runs the request through the Authenticate middleware,
// injecting the userID into the request context exactly as production does.
func withUserID(t *testing.T, r *http.Request, userID string) *http.Request {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  userID,
		"role": "user",
		"exp":  time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(testJWTSecret))
	require.NoError(t, err, "signing JWT")
	r.Header.Set("Authorization", "Bearer "+tok)

	var outCtx context.Context
	middleware.Authenticate(testJWTSecret)(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		outCtx = req.Context()
	})).ServeHTTP(httptest.NewRecorder(), r)

	return r.WithContext(outCtx)
}

// withRole signs a JWT with the given role and runs the request through Authenticate,
// injecting both the userID and role into the request context.
func withRole(t *testing.T, r *http.Request, userID string, role model.Role) *http.Request {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  userID,
		"role": role.String(),
		"exp":  time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(testJWTSecret))
	require.NoError(t, err, "signing JWT")
	r.Header.Set("Authorization", "Bearer "+tok)

	var outCtx context.Context
	middleware.Authenticate(testJWTSecret)(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		outCtx = req.Context()
	})).ServeHTTP(httptest.NewRecorder(), r)

	return r.WithContext(outCtx)
}

// decodeJSON decodes the JSON body from the recorder into v.
func decodeJSON(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	require.NoError(t, json.NewDecoder(w.Body).Decode(v), "decodeJSON")
}

// ============================================================================
// mockAuthService — in-memory stub that implements service.AuthService.
// Set the relevant fields to control what each method returns in a test.
// ============================================================================

type mockAuthService struct {
	loginResult      *service.LoginResult
	loginErr         error
	logoutErr        error
	verifyAccErr     error
	sendResetErr     error
	resetPassErr     error
	sendOTPErr       error
	verifyOTPPair    *service.TokenPair
	verifyOTPErr     error
	refreshPair      *service.TokenPair
	refreshErr       error
	lastRefreshToken string
	sendVerifyErr    error
}

func (m *mockAuthService) Login(_ context.Context, _, _, _, _ string) (*service.LoginResult, error) {
	return m.loginResult, m.loginErr
}

func (m *mockAuthService) Logout(_ context.Context, _ string) error {
	return m.logoutErr
}

func (m *mockAuthService) SendAccountVerification(_ context.Context, _ string) error {
	return m.sendVerifyErr
}

func (m *mockAuthService) VerifyAccount(_ context.Context, _, _ string) error {
	return m.verifyAccErr
}

func (m *mockAuthService) SendPasswordReset(_ context.Context, _ string) error {
	return m.sendResetErr
}

func (m *mockAuthService) ResetPassword(_ context.Context, _, _, _ string) error {
	return m.resetPassErr
}

func (m *mockAuthService) SendOTP(_ context.Context, _ string) error {
	return m.sendOTPErr
}

func (m *mockAuthService) VerifyOTP(_ context.Context, _, _ string) (*service.TokenPair, error) {
	return m.verifyOTPPair, m.verifyOTPErr
}

func (m *mockAuthService) RefreshToken(_ context.Context, token string) (*service.TokenPair, error) {
	m.lastRefreshToken = token
	return m.refreshPair, m.refreshErr
}

// ============================================================================
// mockUserService — in-memory stub that implements service.UserService.
// Set the relevant fields to control what each method returns in a test.
// ============================================================================

type mockUserService struct {
	user            *model.User
	users           []model.User
	getByIDErr      error
	registerErr     error
	updateErr       error
	changePassErr   error
	deleteErr       error
	loginHistory    []model.LoginAuditLog
	loginHistoryErr error
}

func (m *mockUserService) List(_ context.Context, _ store.UserFilter) ([]model.User, int, error) {
	return m.users, len(m.users), nil
}

func (m *mockUserService) GetByID(_ context.Context, _ string) (*model.User, error) {
	return m.user, m.getByIDErr
}

func (m *mockUserService) Register(_ context.Context, u *model.User, _ string) error {
	u.ID = testNewUserID
	return m.registerErr
}

func (m *mockUserService) UpdateProfile(_ context.Context, _ *model.User) error {
	return m.updateErr
}

func (m *mockUserService) ChangePassword(_ context.Context, _, _, _ string) error {
	return m.changePassErr
}

func (m *mockUserService) Delete(_ context.Context, _ string) error {
	return m.deleteErr
}

func (m *mockUserService) LoginHistory(_ context.Context, _ string, _ int) ([]model.LoginAuditLog, error) {
	return m.loginHistory, m.loginHistoryErr
}

// ============================================================================
// mockTagService
// ============================================================================

type mockTagService struct {
	tag       *model.Tag
	tags      []model.Tag
	getErr    error
	createErr error
	updateErr error
	deleteErr error
}

func (m *mockTagService) List(_ context.Context) ([]model.Tag, error) {
	return m.tags, m.getErr
}

func (m *mockTagService) GetByID(_ context.Context, _ string) (*model.Tag, error) {
	return m.tag, m.getErr
}

func (m *mockTagService) GetBySlug(_ context.Context, _ string) (*model.Tag, error) {
	return m.tag, m.getErr
}

func (m *mockTagService) Create(_ context.Context, t *model.Tag) error {
	t.ID = "01966b0a-aaaa-7abc-def0-000000000001"
	return m.createErr
}

func (m *mockTagService) Update(_ context.Context, _ *model.Tag) error {
	return m.updateErr
}

func (m *mockTagService) Delete(_ context.Context, _ string) error {
	return m.deleteErr
}

// ============================================================================
// mockCategoryService
// ============================================================================

type mockCategoryService struct {
	category   *model.Category
	categories []model.Category
	getErr     error
	createErr  error
	updateErr  error
	deleteErr  error
}

func (m *mockCategoryService) List(_ context.Context) ([]model.Category, error) {
	return m.categories, m.getErr
}

func (m *mockCategoryService) GetByID(_ context.Context, _ string) (*model.Category, error) {
	return m.category, m.getErr
}

func (m *mockCategoryService) GetBySlug(_ context.Context, _ string) (*model.Category, error) {
	return m.category, m.getErr
}

func (m *mockCategoryService) Create(_ context.Context, c *model.Category) error {
	c.ID = "01966b0a-bbbb-7abc-def0-000000000002"
	return m.createErr
}

func (m *mockCategoryService) Update(_ context.Context, _ *model.Category) error {
	return m.updateErr
}

func (m *mockCategoryService) Delete(_ context.Context, _ string) error {
	return m.deleteErr
}

// ============================================================================
// mockChapterService
// ============================================================================

type mockChapterService struct {
	chapter   *model.Chapter
	chapters  []model.Chapter
	getErr    error
	createErr error
	updateErr error
	deleteErr error
}

func (m *mockChapterService) List(_ context.Context, _ string) ([]model.Chapter, error) {
	return m.chapters, m.getErr
}

func (m *mockChapterService) GetByID(_ context.Context, _ string) (*model.Chapter, error) {
	return m.chapter, m.getErr
}

func (m *mockChapterService) GetBySlug(_ context.Context, _, _ string) (*model.Chapter, error) {
	return m.chapter, m.getErr
}

func (m *mockChapterService) Create(_ context.Context, c *model.Chapter) error {
	c.ID = "01966b0a-cccc-7abc-def0-000000000003"
	return m.createErr
}

func (m *mockChapterService) Update(_ context.Context, _ *model.Chapter) error {
	return m.updateErr
}

func (m *mockChapterService) Delete(_ context.Context, _ string) error {
	return m.deleteErr
}

// ============================================================================
// mockLessonService
// ============================================================================

type mockLessonService struct {
	lesson    *model.Lesson
	lessons   []model.Lesson
	resource  *model.LessonResource
	resources []model.LessonResource
	getErr    error
	createErr error
	updateErr error
	deleteErr error
}

func (m *mockLessonService) List(_ context.Context, _ string) ([]model.Lesson, error) {
	return m.lessons, m.getErr
}

func (m *mockLessonService) GetByID(_ context.Context, _ string) (*model.Lesson, error) {
	return m.lesson, m.getErr
}

func (m *mockLessonService) GetBySlug(_ context.Context, _, _ string) (*model.Lesson, error) {
	return m.lesson, m.getErr
}

func (m *mockLessonService) Create(_ context.Context, l *model.Lesson) error {
	l.ID = "01966b0a-dddd-7abc-def0-000000000004"
	return m.createErr
}

func (m *mockLessonService) Update(_ context.Context, _ *model.Lesson) error {
	return m.updateErr
}

func (m *mockLessonService) Delete(_ context.Context, _ string) error {
	return m.deleteErr
}

func (m *mockLessonService) ListResources(_ context.Context, _ string) ([]model.LessonResource, error) {
	return m.resources, m.getErr
}

func (m *mockLessonService) GetResourceByID(_ context.Context, _ string) (*model.LessonResource, error) {
	return m.resource, m.getErr
}

func (m *mockLessonService) AddResource(_ context.Context, r *model.LessonResource) error {
	r.ID = "01966b0a-eeee-7abc-def0-000000000005"
	return m.createErr
}

func (m *mockLessonService) UpdateResource(_ context.Context, _ *model.LessonResource) error {
	return m.updateErr
}

func (m *mockLessonService) RemoveResource(_ context.Context, _ string) error {
	return m.deleteErr
}

// ============================================================================
// mockCourseService
// ============================================================================

type mockCourseService struct {
	course           *model.Course
	courses          []model.Course
	getErr           error
	createErr        error
	updateErr        error
	deleteErr        error
	setCategoriesErr error
	setTagsErr       error
}

func (m *mockCourseService) List(_ context.Context, _ store.CourseFilter) ([]model.Course, error) {
	return m.courses, m.getErr
}

func (m *mockCourseService) GetByID(_ context.Context, _ string) (*model.Course, error) {
	return m.course, m.getErr
}

func (m *mockCourseService) GetBySlug(_ context.Context, _ string) (*model.Course, error) {
	return m.course, m.getErr
}

func (m *mockCourseService) Create(_ context.Context, c *model.Course, _ []string, _ string, _ []string) error {
	c.ID = "01966b0a-ffff-7abc-def0-000000000006"
	return m.createErr
}

func (m *mockCourseService) Update(_ context.Context, _ *model.Course) error {
	return m.updateErr
}

func (m *mockCourseService) SetCategories(_ context.Context, _ string, _ []string, _ string) error {
	return m.setCategoriesErr
}

func (m *mockCourseService) SetTags(_ context.Context, _ string, _ []string) error {
	return m.setTagsErr
}

func (m *mockCourseService) Delete(_ context.Context, _ string) error {
	return m.deleteErr
}

// ============================================================================
// mockEnrollmentService
// ============================================================================

type mockEnrollmentService struct {
	enrollment  *model.CourseEnrollment
	enrollments []model.CourseEnrollment
	getErr      error
	enrollErr   error
	updateErr   error
	deleteErr   error
}

func (m *mockEnrollmentService) List(_ context.Context, _ store.EnrollmentFilter) ([]model.CourseEnrollment, error) {
	return m.enrollments, m.getErr
}

func (m *mockEnrollmentService) GetByID(_ context.Context, _ string) (*model.CourseEnrollment, error) {
	return m.enrollment, m.getErr
}

func (m *mockEnrollmentService) Enroll(_ context.Context, userID, courseID string) (*model.CourseEnrollment, error) {
	if m.enrollErr != nil {
		return nil, m.enrollErr
	}
	return &model.CourseEnrollment{
		ID:       "01966b0a-eeee-7abc-def0-000000000099",
		UserID:   userID,
		CourseID: courseID,
		Status:   model.EnrollmentStatusActive,
	}, nil
}

func (m *mockEnrollmentService) UpdateStatus(_ context.Context, _ string, _ model.EnrollmentStatus) error {
	return m.updateErr
}

func (m *mockEnrollmentService) Delete(_ context.Context, _ string) error {
	return m.deleteErr
}
