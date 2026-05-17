// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

// Package router configures the chi router, middleware stack, and all routes.
package router

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	httprateredis "github.com/go-chi/httprate-redis"
	_ "github.com/nanoninja/dojo/docs/swagger" // registers Swagger UI assets
	"github.com/nanoninja/dojo/internal/config"
	"github.com/nanoninja/dojo/internal/handler"
	"github.com/nanoninja/dojo/internal/httputil"
	mw "github.com/nanoninja/dojo/internal/middleware"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/platform/cache"
	"github.com/nanoninja/dojo/internal/platform/database"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Handlers groups all HTTP handlers.
type Handlers struct {
	Auth       *handler.AuthHandler
	User       *handler.UserHandler
	Course     *handler.CourseHandler
	Category   *handler.CategoryHandler
	Tag        *handler.TagHandler
	Chapter    *handler.ChapterHandler
	Lesson     *handler.LessonHandler
	Enrollment *handler.EnrollmentHandler
	Bundle     *handler.BundleHandler
	Progress   *handler.ProgressHandler
}

// New builds and returns the main HTTP router with all middleware and routes configured.
func New(
	handlers *Handlers,
	cfg *config.Config,
	logger *slog.Logger,
	db *database.DB,
	redis *cache.Client,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware — applied to all routes including swagger
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// CSRF
	const (
		csrfCookieName = "csrf_token"
		csrfHeaderName = "X-CSRF-Token"
	)
	csrfEnabled := cfg.AuthTransport.Mode == "cookie" || cfg.AuthTransport.Mode == "dual"

	// Basic CORS
	allowCredentials := cfg.AuthTransport.Mode == "cookie" || cfg.AuthTransport.Mode == "dual"

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: allowCredentials,
		MaxAge:           cfg.CORS.MaxAge,
	}))

	// Swagger UI — dev and test only, registered before SecureHeaders so the
	// browser can load the embedded scripts and styles without CSP restrictions.
	if cfg.App.Env == "development" || cfg.App.Env == "test" {
		r.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
		))
	}

	newRateLimiter := func(limit int, duration time.Duration) func(http.Handler) http.Handler {
		opts := []httprate.Option{httprate.WithKeyFuncs(httprate.KeyByIP)}
		if redis != nil {
			store, _ := httprateredis.NewRedisLimitCounter(&httprateredis.Config{
				Client: redis.Client,
			})
			opts = append(opts, httprate.WithLimitCounter(store))
		}
		return httprate.Limit(limit, duration, opts...)
	}

	health := handler.NewHealthHandler(cfg.App.Version, cfg.App.Env, db, redis)

	// All routes below carry strict security headers and body size limit.
	r.Group(func(r chi.Router) {
		r.Use(mw.SecureHeaders(cfg.App.Env))
		r.Use(mw.MaxBodySize(1 * 1024 * 1024)) // 1 MB
		r.Use(mw.PrometheusMetrics)

		r.Get("/health", httputil.Handle(health.Health, logger))
		r.Get("/livez", httputil.Handle(health.Live, logger))
		r.Get("/readyz", httputil.Handle(health.Ready, logger))

		r.Group(func(r chi.Router) {
			if len(cfg.App.MetricsAllowedIPs) > 0 {
				r.Use(mw.IPAllowList(cfg.App.MetricsAllowedIPs))
			}
			r.Get("/metrics", promhttp.Handler().ServeHTTP)
		})

		// Public auth routes — 3 requests/min
		r.Group(func(r chi.Router) {
			r.Use(newRateLimiter(3, 1*time.Minute))

			r.Post("/auth/register", httputil.Handle(handlers.Auth.Register, logger))
			r.Post("/auth/password/reset", httputil.Handle(handlers.Auth.SendPasswordReset, logger))
			r.Post("/auth/password/new", httputil.Handle(handlers.Auth.ResetPassword, logger))
		})

		// Public auth routes — 5 requests/min
		r.Group(func(r chi.Router) {
			r.Use(newRateLimiter(5, 1*time.Minute))

			r.Post("/auth/login", httputil.Handle(handlers.Auth.Login, logger))
			r.Post("/auth/verify", httputil.Handle(handlers.Auth.VerifyAccount, logger))
			r.Post("/auth/verify/resend", httputil.Handle(handlers.Auth.ResendVerification, logger))
			r.Post("/auth/otp/verify", httputil.Handle(handlers.Auth.VerifyOTP, logger))
			r.Post("/auth/otp/resend", httputil.Handle(handlers.Auth.ResendOTP, logger))
		})

		r.Group(func(r chi.Router) {
			r.Use(newRateLimiter(5, 1*time.Minute))
			r.Use(mw.RequireCSRF(csrfEnabled, csrfCookieName, csrfHeaderName))

			r.Post("/auth/token/refresh", httputil.Handle(handlers.Auth.RefreshToken, logger))
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(mw.AuthenticateWithTransport(
				cfg.JWT.Secret,
				cfg.AuthTransport.Mode,
				cfg.AuthTransport.AccessCookieName,
			))
			r.Use(mw.RequireCSRF(csrfEnabled, csrfCookieName, csrfHeaderName))

			r.Post("/auth/logout", httputil.Handle(handlers.Auth.Logout, logger))

			r.Route("/api/v1", func(r chi.Router) {

				// Users
				r.Get("/users/me", httputil.Handle(handlers.User.Me, logger))
				r.Get("/users/me/login-history", httputil.Handle(handlers.User.LoginHistory, logger))
				r.Put("/users/{id}/profile", httputil.Handle(handlers.User.UpdateProfile, logger))
				r.Put("/users/{id}/password", httputil.Handle(handlers.User.ChangePassword, logger))

				r.Group(func(r chi.Router) {
					r.Use(mw.RequireRole(model.RoleAdmin))

					r.Get("/users", httputil.Handle(handlers.User.List, logger))
					r.Get("/users/{id}", httputil.Handle(handlers.User.GetByID, logger))
					r.Delete("/users/{id}", httputil.Handle(handlers.User.Delete, logger))
				})

				// Courses — read
				r.Get("/courses", httputil.Handle(handlers.Course.List, logger))
				r.Get("/courses/{id}", httputil.Handle(handlers.Course.GetByID, logger))
				r.Get("/courses/{course_id}/chapters", httputil.Handle(handlers.Chapter.List, logger))

				// Categories — read
				r.Get("/categories", httputil.Handle(handlers.Category.List, logger))
				r.Get("/categories/{id}", httputil.Handle(handlers.Category.GetByID, logger))

				// Tags — read
				r.Get("/tags", httputil.Handle(handlers.Tag.List, logger))
				r.Get("/tags/slug/{slug}", httputil.Handle(handlers.Tag.GetBySlug, logger))
				r.Get("/tags/{id}", httputil.Handle(handlers.Tag.GetByID, logger))

				// Chapters — read
				r.Get("/chapters/{id}", httputil.Handle(handlers.Chapter.GetByID, logger))
				r.Get("/chapters/{chapter_id}/lessons", httputil.Handle(handlers.Lesson.List, logger))

				// Lessons — read
				r.Get("/lessons/{id}", httputil.Handle(handlers.Lesson.GetByID, logger))
				r.Get("/lessons/{id}/resources", httputil.Handle(handlers.Lesson.ListResources, logger))

				// Enrollments
				r.Get("/enrollments", httputil.Handle(handlers.Enrollment.List, logger))
				r.Get("/enrollments/{id}", httputil.Handle(handlers.Enrollment.GetByID, logger))
				r.Post("/enrollments", httputil.Handle(handlers.Enrollment.Enroll, logger))
				r.Patch("/enrollments/{id}/status", httputil.Handle(handlers.Enrollment.UpdateStatus, logger))
				r.Delete("/enrollments/{id}", httputil.Handle(handlers.Enrollment.Delete, logger))

				// Progress
				r.Get("/progress/{user_id}/lessons/{lesson_id}", httputil.Handle(handlers.Progress.Get, logger))
				r.Get("/progress/{user_id}/courses/course_id", httputil.Handle(handlers.Progress.ListByCourse, logger))
				r.Post("/progress", httputil.Handle(handlers.Progress.Save, logger))

				// Bundles — read
				r.Get("/bundles", httputil.Handle(handlers.Bundle.List, logger))
				r.Get("/bundles/{id}", httputil.Handle(handlers.Bundle.GetByID, logger))

				// Instructor+ — gestion des cours, chapitres, leçons
				r.Group(func(r chi.Router) {
					r.Use(mw.RequireRole(model.RoleInstructor))

					r.Post("/courses", httputil.Handle(handlers.Course.Create, logger))
					r.Put("/courses/{id}", httputil.Handle(handlers.Course.Update, logger))
					r.Delete("/courses/{id}", httputil.Handle(handlers.Course.Delete, logger))
					r.Put("/courses/{id}/categories", httputil.Handle(handlers.Course.SetCategories, logger))
					r.Put("/courses/{id}/tags", httputil.Handle(handlers.Course.SetTags, logger))

					r.Post("/chapters", httputil.Handle(handlers.Chapter.Create, logger))
					r.Put("/chapters/{id}", httputil.Handle(handlers.Chapter.Update, logger))
					r.Delete("/chapters/{id}", httputil.Handle(handlers.Chapter.Delete, logger))

					r.Post("/lessons", httputil.Handle(handlers.Lesson.Create, logger))
					r.Put("/lessons/{id}", httputil.Handle(handlers.Lesson.Update, logger))
					r.Delete("/lessons/{id}", httputil.Handle(handlers.Lesson.Delete, logger))
					r.Post("/lessons/{id}/resources", httputil.Handle(handlers.Lesson.AddResource, logger))
					r.Put("/lessons/resources/{id}", httputil.Handle(handlers.Lesson.UpdateResource, logger))
					r.Delete("/lessons/resources/{id}", httputil.Handle(handlers.Lesson.RemoveResource, logger))

					r.Post("/bundles", httputil.Handle(handlers.Bundle.Create, logger))
					r.Put("/bundles/{id}", httputil.Handle(handlers.Bundle.Update, logger))
					r.Put("/bundles/{id}/courses", httputil.Handle(handlers.Bundle.SetCourses, logger))
					r.Delete("/bundles/{id}", httputil.Handle(handlers.Bundle.Delete, logger))
				})

				// Admin — gestion des catégories et tags
				r.Group(func(r chi.Router) {
					r.Use(mw.RequireRole(model.RoleAdmin))

					r.Post("/categories", httputil.Handle(handlers.Category.Create, logger))
					r.Put("/categories/{id}", httputil.Handle(handlers.Category.Update, logger))
					r.Delete("/categories/{id}", httputil.Handle(handlers.Category.Delete, logger))

					r.Post("/tags", httputil.Handle(handlers.Tag.Create, logger))
					r.Put("/tags/{id}", httputil.Handle(handlers.Tag.Update, logger))
					r.Delete("/tags/{id}", httputil.Handle(handlers.Tag.Delete, logger))
				})
			})
		})
	})

	return r
}
