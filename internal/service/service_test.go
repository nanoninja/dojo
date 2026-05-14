// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package service_test

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/platform/database"
	"github.com/nanoninja/dojo/internal/store"
)

// ============================================================================
// fakeUserStore
// ============================================================================

type fakeUserStore struct {
	users   map[string]*model.User // by ID
	byEmail map[string]*model.User // by email
	seq     int
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{
		users:   make(map[string]*model.User),
		byEmail: make(map[string]*model.User),
	}
}

func (f *fakeUserStore) nextID() string {
	f.seq++
	return fmt.Sprintf("user-%d", f.seq)
}

func (f *fakeUserStore) List(_ context.Context, _ store.UserFilter) ([]model.User, int, error) {
	var result []model.User
	for _, u := range f.users {
		if u.Status != model.UserStatusDeleted {
			result = append(result, *u)
		}
	}
	return result, len(result), nil
}

func (f *fakeUserStore) FindByID(_ context.Context, id string) (*model.User, error) {
	u, ok := f.users[id]
	if !ok || u.Status == model.UserStatusDeleted {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (f *fakeUserStore) FindByEmail(_ context.Context, email string) (*model.User, error) {
	u, ok := f.byEmail[email]
	if !ok || u.Status == model.UserStatusDeleted {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (f *fakeUserStore) FindCredentialsByID(_ context.Context, id string) (*model.User, error) {
	u, err := f.FindByID(context.TODO(), id)
	return u, err
}

func (f *fakeUserStore) Create(_ context.Context, u *model.User) error {
	u.ID = f.nextID()
	cp := *u
	f.users[u.ID] = &cp
	f.byEmail[u.Email] = &cp
	return nil
}

func (f *fakeUserStore) Update(_ context.Context, u *model.User) error {
	if _, ok := f.users[u.ID]; !ok {
		return fmt.Errorf("user not found")
	}
	cp := *u
	f.users[u.ID] = &cp
	f.byEmail[u.Email] = &cp
	return nil
}

func (f *fakeUserStore) UpdatePassword(_ context.Context, id, hash string) error {
	u, ok := f.users[id]
	if !ok {
		return fmt.Errorf("user not found")
	}
	u.PasswordHash = hash
	return nil
}

func (f *fakeUserStore) UpdateLastLogin(_ context.Context, id, _ string) error {
	u, ok := f.users[id]
	if !ok {
		return fmt.Errorf("user not found")
	}
	now := time.Now()
	u.LastLoginAt = &now
	u.LoginCount++
	return nil
}

func (f *fakeUserStore) UpdateVerified(_ context.Context, id string) error {
	u, ok := f.users[id]
	if !ok {
		return fmt.Errorf("user not found")
	}
	u.IsVerified = true
	return nil
}

func (f *fakeUserStore) Delete(_ context.Context, id string) error {
	u, ok := f.users[id]
	if !ok {
		return fmt.Errorf("user not found")
	}
	u.Status = model.UserStatusDeleted
	return nil
}

func (f *fakeUserStore) IncrementFailedLogin(_ context.Context, id string) error {
	if u, ok := f.users[id]; ok {
		u.FailedLoginAttempts++
	}
	return nil
}

func (f *fakeUserStore) LockAccount(_ context.Context, id string, until time.Time) error {
	if u, ok := f.users[id]; ok {
		u.LockedUntil = &until
	}
	return nil
}

func (f *fakeUserStore) ResetFailedLogin(_ context.Context, id string) error {
	if u, ok := f.users[id]; ok {
		u.FailedLoginAttempts = 0
		u.LockedUntil = nil
	}
	return nil
}

// ============================================================================
// fakeAuthStore
// ============================================================================

type fakeAuthStore struct {
	tokens map[string]*model.VerificationToken // by ID
	seq    int
}

func newFakeAuthStore() *fakeAuthStore {
	return &fakeAuthStore{tokens: make(map[string]*model.VerificationToken)}
}

func (f *fakeAuthStore) nextID() string {
	f.seq++
	return fmt.Sprintf("token-%d", f.seq)
}

func (f *fakeAuthStore) Create(_ context.Context, t *model.VerificationToken) error {
	t.ID = f.nextID()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	cp := *t
	f.tokens[t.ID] = &cp
	return nil
}

func (f *fakeAuthStore) FindActiveByUserAndType(_ context.Context, userID string, tokenType model.TokenType) (*model.VerificationToken, error) {
	var latest *model.VerificationToken

	for _, t := range f.tokens {
		// Keep only active tokens for this user and token type.
		if t.UserID != userID || t.Type != tokenType {
			continue
		}
		if t.UsedAt != nil || !t.ExpiresAt.After(time.Now()) {
			continue
		}

		// Return the most recently created active token.
		if latest == nil || t.CreatedAt.After(latest.CreatedAt) {
			tt := *t
			latest = &tt
		}
	}

	if latest == nil {
		return nil, nil
	}
	return latest, nil
}

func (f *fakeAuthStore) FindOne(_ context.Context, filter store.TokenFilter) (*model.VerificationToken, error) {
	for _, t := range f.tokens {
		if t.UserID == filter.UserID &&
			t.Token == filter.Token &&
			t.Type == filter.Type &&
			t.UsedAt == nil &&
			t.ExpiresAt.After(time.Now()) {
			cp := *t
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *fakeAuthStore) MarkUsed(_ context.Context, id string) error {
	t, ok := f.tokens[id]
	if !ok {
		return fmt.Errorf("token not found")
	}
	now := time.Now()
	t.UsedAt = &now
	return nil
}

func (f *fakeAuthStore) IncrementAttempts(_ context.Context, id string) error {
	t, ok := f.tokens[id]
	if !ok {
		return fmt.Errorf("token not found")
	}
	t.Attempts++
	return nil
}

func (f *fakeAuthStore) DeleteExpired(_ context.Context, userID string) error {
	for id, t := range f.tokens {
		if t.UserID == userID && t.ExpiresAt.Before(time.Now()) {
			delete(f.tokens, id)
		}
	}
	return nil
}

// ============================================================================
// fakeRefreshTokenStore
// ============================================================================

type fakeRefreshTokenStore struct {
	tokens     map[string]*model.RefreshToken // by hash
	seq        int
	failRotate bool // when true, RotateToken returns an error without modifying state
}

func newFakeRefreshTokenStore() *fakeRefreshTokenStore {
	return &fakeRefreshTokenStore{tokens: make(map[string]*model.RefreshToken)}
}

func (f *fakeRefreshTokenStore) nextID() string {
	f.seq++
	return fmt.Sprintf("rt-%d", f.seq)
}

func (f *fakeRefreshTokenStore) Create(_ context.Context, t *model.RefreshToken) error {
	t.ID = f.nextID()
	cp := *t
	f.tokens[t.TokenHash] = &cp
	return nil
}

func (f *fakeRefreshTokenStore) FindByHash(_ context.Context, hash string) (*model.RefreshToken, error) {
	t, ok := f.tokens[hash]
	if !ok || t.RevokedAt != nil || t.ExpiresAt.Before(time.Now()) {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}

func (f *fakeRefreshTokenStore) Revoke(_ context.Context, id string) error {
	for _, t := range f.tokens {
		if t.ID == id {
			now := time.Now()
			t.RevokedAt = &now
			return nil
		}
	}
	return fmt.Errorf("token not found")
}

func (f *fakeRefreshTokenStore) RevokeAllForUser(_ context.Context, userID string) error {
	now := time.Now()
	for _, t := range f.tokens {
		if t.UserID == userID && t.RevokedAt == nil {
			t.RevokedAt = &now
		}
	}
	return nil
}

func (f *fakeRefreshTokenStore) RotateToken(ctx context.Context, oldID string, newToken *model.RefreshToken) error {
	if f.failRotate {
		return fmt.Errorf("simulated RotateToken failure")
	}
	if err := f.Revoke(ctx, oldID); err != nil {
		return err
	}
	return f.Create(ctx, newToken)
}

func (f *fakeRefreshTokenStore) DeleteExpired(_ context.Context, userID string) error {
	for hash, t := range f.tokens {
		if t.UserID == userID && t.ExpiresAt.Before(time.Now()) {
			delete(f.tokens, hash)
		}
	}
	return nil
}

// ============================================================================
// fakeMailer
// ============================================================================

type fakeMailer struct {
	sentVerification []string
	sentReset        []string
	sentOTP          []string
}

func (f *fakeMailer) SendAccountVerification(_ context.Context, to, _ string) error {
	f.sentVerification = append(f.sentVerification, to)
	return nil
}

func (f *fakeMailer) SendPasswordReset(_ context.Context, to, _ string) error {
	f.sentReset = append(f.sentReset, to)
	return nil
}

func (f *fakeMailer) SendOTP(_ context.Context, to, _ string) error {
	f.sentOTP = append(f.sentOTP, to)
	return nil
}

// ============================================================================
// fakeLoginAuditStore
// ============================================================================

type fakeLoginAuditStore struct {
	logs []model.LoginAuditLog
}

func (f *fakeLoginAuditStore) Create(_ context.Context, log *model.LoginAuditLog) error {
	f.logs = append(f.logs, *log)
	return nil
}

func (f *fakeLoginAuditStore) FindByUser(_ context.Context, _ string, _ int) ([]model.LoginAuditLog, error) {
	return f.logs, nil
}

func (f *fakeLoginAuditStore) List(_ context.Context, _ store.AuditFilter) ([]model.LoginAuditLog, int, error) {
	return f.logs, len(f.logs), nil
}

func (f *fakeLoginAuditStore) Purge(_ context.Context, _ time.Duration, _ int) (int64, error) {
	return 0, nil
}

// ============================================================================
// fakeTagStore
// ============================================================================

type fakeTagStore struct {
	tags map[string]*model.Tag
	seq  int
}

func newFakeTagStore() *fakeTagStore {
	return &fakeTagStore{tags: make(map[string]*model.Tag)}
}

func (f *fakeTagStore) nextID() string {
	f.seq++
	return fmt.Sprintf("tag-%d", f.seq)
}

func (f *fakeTagStore) List(_ context.Context) ([]model.Tag, error) {
	result := make([]model.Tag, 0, len(f.tags))
	for _, t := range f.tags {
		result = append(result, *t)
	}
	return result, nil
}

func (f *fakeTagStore) FindByID(_ context.Context, id string) (*model.Tag, error) {
	t, ok := f.tags[id]
	if !ok {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}

func (f *fakeTagStore) FindBySlug(_ context.Context, slug string) (*model.Tag, error) {
	for _, t := range f.tags {
		if t.Slug == slug {
			cp := *t
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *fakeTagStore) Create(_ context.Context, t *model.Tag) error {
	t.ID = f.nextID()
	cp := *t
	f.tags[t.ID] = &cp
	return nil
}

func (f *fakeTagStore) Update(_ context.Context, t *model.Tag) error {
	if _, ok := f.tags[t.ID]; !ok {
		return fmt.Errorf("tag not found")
	}
	cp := *t
	f.tags[t.ID] = &cp
	return nil
}

func (f *fakeTagStore) Delete(_ context.Context, id string) error {
	if _, ok := f.tags[id]; !ok {
		return fmt.Errorf("tag not found")
	}
	delete(f.tags, id)
	return nil
}

// ============================================================================
// fakeCategoryStore
// ============================================================================

type fakeCategoryStore struct {
	categories map[string]*model.Category
	seq        int
}

func newFakeCategoryStore() *fakeCategoryStore {
	return &fakeCategoryStore{categories: make(map[string]*model.Category)}
}

func (f *fakeCategoryStore) nextID() string {
	f.seq++
	return fmt.Sprintf("cat-%d", f.seq)
}

func (f *fakeCategoryStore) List(_ context.Context) ([]model.Category, error) {
	result := make([]model.Category, 0, len(f.categories))
	for _, c := range f.categories {
		if c.DeletedAt == nil {
			result = append(result, *c)
		}
	}
	return result, nil
}

func (f *fakeCategoryStore) FindByID(_ context.Context, id string) (*model.Category, error) {
	c, ok := f.categories[id]
	if !ok || c.DeletedAt != nil {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (f *fakeCategoryStore) FindBySlug(_ context.Context, slug string) (*model.Category, error) {
	for _, c := range f.categories {
		if c.Slug == slug && c.DeletedAt == nil {
			cp := *c
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *fakeCategoryStore) Create(_ context.Context, c *model.Category) error {
	c.ID = f.nextID()
	cp := *c
	f.categories[c.ID] = &cp
	return nil
}

func (f *fakeCategoryStore) Update(_ context.Context, c *model.Category) error {
	if _, ok := f.categories[c.ID]; !ok {
		return fmt.Errorf("category not found")
	}
	cp := *c
	f.categories[c.ID] = &cp
	return nil
}

func (f *fakeCategoryStore) Delete(_ context.Context, id string) error {
	c, ok := f.categories[id]
	if !ok {
		return fmt.Errorf("category not found")
	}
	now := time.Now()
	c.DeletedAt = &now
	return nil
}

// ============================================================================
// fakeChapterStore
// ============================================================================

type fakeChapterStore struct {
	chapters map[string]*model.Chapter
	seq      int
}

func newFakeChapterStore() *fakeChapterStore {
	return &fakeChapterStore{chapters: make(map[string]*model.Chapter)}
}

func (f *fakeChapterStore) nextID() string {
	f.seq++
	return fmt.Sprintf("chapter-%d", f.seq)
}

func (f *fakeChapterStore) List(_ context.Context, courseID string) ([]model.Chapter, error) {
	result := make([]model.Chapter, 0)
	for _, c := range f.chapters {
		if c.CourseID == courseID {
			result = append(result, *c)
		}
	}
	return result, nil
}

func (f *fakeChapterStore) FindByID(_ context.Context, id string) (*model.Chapter, error) {
	c, ok := f.chapters[id]
	if !ok {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (f *fakeChapterStore) FindBySlug(_ context.Context, courseID, slug string) (*model.Chapter, error) {
	for _, c := range f.chapters {
		if c.CourseID == courseID && c.Slug == slug {
			cp := *c
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *fakeChapterStore) Create(_ context.Context, c *model.Chapter) error {
	c.ID = f.nextID()
	cp := *c
	f.chapters[c.ID] = &cp
	return nil
}

func (f *fakeChapterStore) Update(_ context.Context, c *model.Chapter) error {
	if _, ok := f.chapters[c.ID]; !ok {
		return fmt.Errorf("chapter not found")
	}
	cp := *c
	f.chapters[c.ID] = &cp
	return nil
}

func (f *fakeChapterStore) Delete(_ context.Context, id string) error {
	if _, ok := f.chapters[id]; !ok {
		return fmt.Errorf("chapter not found")
	}
	delete(f.chapters, id)
	return nil
}

// ============================================================================
// fakeLessonStore
// ============================================================================

type fakeLessonStore struct {
	lessons map[string]*model.Lesson
	seq     int
}

func newFakeLessonStore() *fakeLessonStore {
	return &fakeLessonStore{lessons: make(map[string]*model.Lesson)}
}

func (f *fakeLessonStore) nextID() string {
	f.seq++
	return fmt.Sprintf("lesson-%d", f.seq)
}

func (f *fakeLessonStore) List(_ context.Context, chapterID string) ([]model.Lesson, error) {
	result := make([]model.Lesson, 0)
	for _, l := range f.lessons {
		if l.ChapterID == chapterID {
			result = append(result, *l)
		}
	}
	return result, nil
}

func (f *fakeLessonStore) FindByID(_ context.Context, id string) (*model.Lesson, error) {
	l, ok := f.lessons[id]
	if !ok {
		return nil, nil
	}
	cp := *l
	return &cp, nil
}

func (f *fakeLessonStore) FindBySlug(_ context.Context, chapterID, slug string) (*model.Lesson, error) {
	for _, l := range f.lessons {
		if l.ChapterID == chapterID && l.Slug == slug {
			cp := *l
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *fakeLessonStore) Create(_ context.Context, l *model.Lesson) error {
	l.ID = f.nextID()
	cp := *l
	f.lessons[l.ID] = &cp
	return nil
}

func (f *fakeLessonStore) Update(_ context.Context, l *model.Lesson) error {
	if _, ok := f.lessons[l.ID]; !ok {
		return fmt.Errorf("lesson not found")
	}
	cp := *l
	f.lessons[l.ID] = &cp
	return nil
}

func (f *fakeLessonStore) Delete(_ context.Context, id string) error {
	if _, ok := f.lessons[id]; !ok {
		return fmt.Errorf("lesson not found")
	}
	delete(f.lessons, id)
	return nil
}

// ============================================================================
// fakeLessonResourceStore
// ============================================================================

type fakeLessonResourceStore struct {
	resources map[string]*model.LessonResource
	seq       int
}

func newFakeLessonResourceStore() *fakeLessonResourceStore {
	return &fakeLessonResourceStore{resources: make(map[string]*model.LessonResource)}
}

func (f *fakeLessonResourceStore) nextID() string {
	f.seq++
	return fmt.Sprintf("res-%d", f.seq)
}

func (f *fakeLessonResourceStore) List(_ context.Context, lessonID string) ([]model.LessonResource, error) {
	result := make([]model.LessonResource, 0)
	for _, r := range f.resources {
		if r.LessonID == lessonID {
			result = append(result, *r)
		}
	}
	return result, nil
}

func (f *fakeLessonResourceStore) FindByID(_ context.Context, id string) (*model.LessonResource, error) {
	r, ok := f.resources[id]
	if !ok {
		return nil, nil
	}
	cp := *r
	return &cp, nil
}

func (f *fakeLessonResourceStore) Create(_ context.Context, r *model.LessonResource) error {
	r.ID = f.nextID()
	cp := *r
	f.resources[r.ID] = &cp
	return nil
}

func (f *fakeLessonResourceStore) Update(_ context.Context, r *model.LessonResource) error {
	if _, ok := f.resources[r.ID]; !ok {
		return fmt.Errorf("resource not found")
	}
	cp := *r
	f.resources[r.ID] = &cp
	return nil
}

func (f *fakeLessonResourceStore) Delete(_ context.Context, id string) error {
	if _, ok := f.resources[id]; !ok {
		return fmt.Errorf("resource not found")
	}
	delete(f.resources, id)
	return nil
}

// ============================================================================
// fakeCourseStore
// ============================================================================

type fakeCourseStore struct {
	courses map[string]*model.Course
	seq     int
}

func newFakeCourseStore() *fakeCourseStore {
	return &fakeCourseStore{courses: make(map[string]*model.Course)}
}

func (f *fakeCourseStore) nextID() string {
	f.seq++
	return fmt.Sprintf("course-%d", f.seq)
}

func (f *fakeCourseStore) List(_ context.Context, _ store.CourseFilter) ([]model.Course, error) {
	result := make([]model.Course, 0, len(f.courses))
	for _, c := range f.courses {
		if c.DeletedAt == nil {
			result = append(result, *c)
		}
	}
	return result, nil
}

func (f *fakeCourseStore) FindByID(_ context.Context, id string) (*model.Course, error) {
	c, ok := f.courses[id]
	if !ok || c.DeletedAt != nil {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (f *fakeCourseStore) FindBySlug(_ context.Context, slug string) (*model.Course, error) {
	for _, c := range f.courses {
		if c.Slug == slug && c.DeletedAt == nil {
			cp := *c
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *fakeCourseStore) Create(_ context.Context, c *model.Course) error {
	c.ID = f.nextID()
	cp := *c
	f.courses[c.ID] = &cp
	return nil
}

func (f *fakeCourseStore) Update(_ context.Context, c *model.Course) error {
	if _, ok := f.courses[c.ID]; !ok {
		return fmt.Errorf("course not found")
	}
	cp := *c
	f.courses[c.ID] = &cp
	return nil
}

func (f *fakeCourseStore) Delete(_ context.Context, id string) error {
	c, ok := f.courses[id]
	if !ok {
		return fmt.Errorf("course not found")
	}
	now := time.Now()
	c.DeletedAt = &now
	return nil
}

// ============================================================================
// fakeCoursesCategoriesStore
// ============================================================================

type fakeCoursesCategoriesStore struct {
	assignments []model.CategoryAssignment
}

func (f *fakeCoursesCategoriesStore) List(_ context.Context, courseID string) ([]model.CategoryAssignment, error) {
	result := make([]model.CategoryAssignment, 0)
	for _, a := range f.assignments {
		if a.CourseID == courseID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (f *fakeCoursesCategoriesStore) Assign(_ context.Context, courseID, categoryID string, isPrimary bool) error {
	f.assignments = append(f.assignments, model.CategoryAssignment{
		CourseID:   courseID,
		CategoryID: categoryID,
		IsPrimary:  isPrimary,
	})
	return nil
}

func (f *fakeCoursesCategoriesStore) Unassign(_ context.Context, courseID, categoryID string) error {
	result := f.assignments[:0]
	for _, a := range f.assignments {
		if !(a.CourseID == courseID && a.CategoryID == categoryID) {
			result = append(result, a)
		}
	}
	f.assignments = result
	return nil
}

func (f *fakeCoursesCategoriesStore) SetPrimary(_ context.Context, courseID, categoryID string) error {
	for i := range f.assignments {
		if f.assignments[i].CourseID == courseID {
			f.assignments[i].IsPrimary = f.assignments[i].CategoryID == categoryID
		}
	}
	return nil
}

// ============================================================================
// fakeCoursesTagsStore
// ============================================================================

type fakeCoursesTagsStore struct {
	assignments []model.CourseTagAssignment
}

func (f *fakeCoursesTagsStore) List(_ context.Context, courseID string) ([]model.CourseTagAssignment, error) {
	result := make([]model.CourseTagAssignment, 0)
	for _, a := range f.assignments {
		if a.CourseID == courseID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (f *fakeCoursesTagsStore) Assign(_ context.Context, courseID, tagID string) error {
	f.assignments = append(f.assignments, model.CourseTagAssignment{CourseID: courseID, TagID: tagID})
	return nil
}

func (f *fakeCoursesTagsStore) Unassign(_ context.Context, courseID, tagID string) error {
	result := f.assignments[:0]
	for _, a := range f.assignments {
		if !(a.CourseID == courseID && a.TagID == tagID) {
			result = append(result, a)
		}
	}
	f.assignments = result
	return nil
}

// ============================================================================
// fakeTxRunner
// ============================================================================

// fakeTxRunner simulates WithTx by calling fn with a no-op querier.
// The real stores created inside fn will issue no-op SQL calls.
// This lets us test error propagation and happy-path flow without a real DB.
type fakeTxRunner struct {
	err error // when set, WithTx returns this error without calling fn
}

func (f *fakeTxRunner) WithTx(_ context.Context, fn func(database.Querier) error) error {
	if f.err != nil {
		return f.err
	}
	return fn(noopQuerier{})
}

// noopQuerier satisfies database.Querier with no-ops.
// Stores created inside WithTx will succeed without touching a real DB.
type noopQuerier struct{}

func (noopQuerier) GetContext(_ context.Context, _ any, _ string, _ ...any) error   { return nil }
func (noopQuerier) SelectContext(_ context.Context, _ any, _ string, _ ...any) error { return nil }
func (noopQuerier) QueryxContext(_ context.Context, _ string, _ ...any) (*sqlx.Rows, error) {
	return nil, nil
}
func (noopQuerier) QueryRowContext(_ context.Context, _ string, _ ...any) *sql.Row { return nil }
func (noopQuerier) ExecContext(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	return nil, nil
}
func (noopQuerier) Rebind(query string) string { return query }
