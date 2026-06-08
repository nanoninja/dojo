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
	Auth          *handler.AuthHandler
	User          *handler.UserHandler
	Course        *handler.CourseHandler
	Review        *handler.ReviewHandler
	Category      *handler.CategoryHandler
	Tag           *handler.TagHandler
	Chapter       *handler.ChapterHandler
	Lesson        *handler.LessonHandler
	Enrollment    *handler.EnrollmentHandler
	Bundle        *handler.BundleHandler
	Progress      *handler.ProgressHandler
	Certificate   *handler.CertificateHandler
	Consent       *handler.ConsentHandler
	Subscriptions *handler.SubscriptionHandler
	Purchase      *handler.PurchaseHandler
	Webhook       *handler.WebhookHandler
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

	// newUserRateLimiter limits by authenticated userID when available, falls back to IP.
	// Used on mutation endpoints to prevent abuse from compromised tokens.
	newUserRateLimiter := func(limit int, duration time.Duration) func(http.Handler) http.Handler {
		opts := []httprate.Option{
			httprate.WithKeyFuncs(func(r *http.Request) (string, error) {
				if id := mw.UserIDFromContext(r.Context()); id != "" {
					return "user:" + id, nil
				}
				return httprate.KeyByRealIP(r)
			}),
		}
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

		r.Get("/courses/{course_id}/reviews", httputil.Handle(handlers.Review.List, logger))
		r.Get("/courses/{course_id}/reviews/{id}", httputil.Handle(handlers.Review.GetByID, logger))
		r.Get("/certificates/verify/{uuid}", httputil.Handle(handlers.Certificate.Verify, logger))

		// Stripe webhook — public, no auth, signature validated by handler
		r.Post("/webhooks/stripe", httputil.Handle(handlers.Webhook.Stripe, logger))

		// ── Authenticated ─────────────────────────────────────────────────────────────

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
				r.Route("/users", func(r chi.Router) {
					r.Get("/me", httputil.Handle(handlers.User.Me, logger))
					r.Get("/me/login-history", httputil.Handle(handlers.User.LoginHistory, logger))
					r.Put("/{id}/profile", httputil.Handle(handlers.User.UpdateProfile, logger))
					r.Put("/{id}/password", httputil.Handle(handlers.User.ChangePassword, logger))

					r.Group(func(r chi.Router) {
						r.Use(mw.RequireRole(model.RoleAdmin))
						r.Get("/", httputil.Handle(handlers.User.List, logger))
						r.Get("/{id}", httputil.Handle(handlers.User.GetByID, logger))
						r.Delete("/{id}", httputil.Handle(handlers.User.Delete, logger))
					})
				})

				// Courses & Reviews
				r.Route("/courses", func(r chi.Router) {
					r.Use(newRateLimiter(60, time.Minute))

					r.Get("/", httputil.Handle(handlers.Course.List, logger))
					r.Get("/{id}", httputil.Handle(handlers.Course.GetByID, logger))
					r.Get("/{course_id}/chapters", httputil.Handle(handlers.Chapter.List, logger))

					r.Group(func(r chi.Router) {
						r.Use(mw.RequireRole(model.RoleInstructor))

						r.Post("/", httputil.Handle(handlers.Course.Create, logger))
						r.Put("/{id}", httputil.Handle(handlers.Course.Update, logger))
						r.Delete("/{id}", httputil.Handle(handlers.Course.Delete, logger))
						r.Put("/{id}/categories", httputil.Handle(handlers.Course.SetCategories, logger))
						r.Put("/{id}/tags", httputil.Handle(handlers.Course.SetTags, logger))
						r.Post("/{course_id}/reviews", httputil.Handle(handlers.Review.Create, logger))
						r.Put("/{course_id}/reviews/{id}", httputil.Handle(handlers.Review.Update, logger))
						r.Delete("/{course_id}/reviews/{id}", httputil.Handle(handlers.Review.Delete, logger))
					})
				})

				// Categories
				r.Route("/categories", func(r chi.Router) {
					r.Get("/", httputil.Handle(handlers.Category.List, logger))
					r.Get("/{id}", httputil.Handle(handlers.Category.GetByID, logger))

					r.Group(func(r chi.Router) {
						r.Use(mw.RequireRole(model.RoleAdmin))

						r.Post("/", httputil.Handle(handlers.Category.Create, logger))
						r.Put("/{id}", httputil.Handle(handlers.Category.Update, logger))
						r.Delete("/{id}", httputil.Handle(handlers.Category.Delete, logger))
					})
				})

				// Tags
				r.Route("/tags", func(r chi.Router) {
					r.Get("/", httputil.Handle(handlers.Tag.List, logger))
					r.Get("/slug/{slug}", httputil.Handle(handlers.Tag.GetBySlug, logger))
					r.Get("/{id}", httputil.Handle(handlers.Tag.GetByID, logger))

					r.Group(func(r chi.Router) {
						r.Use(mw.RequireRole(model.RoleAdmin))

						r.Post("/", httputil.Handle(handlers.Tag.Create, logger))
						r.Put("/{id}", httputil.Handle(handlers.Tag.Update, logger))
						r.Delete("/{id}", httputil.Handle(handlers.Tag.Delete, logger))
					})
				})

				// Chapters
				r.Route("/chapters", func(r chi.Router) {
					r.Get("/{id}", httputil.Handle(handlers.Chapter.GetByID, logger))
					r.Get("/{chapter_id}/lessons", httputil.Handle(handlers.Lesson.List, logger))

					r.Group(func(r chi.Router) {
						r.Use(mw.RequireRole(model.RoleInstructor))
						r.Post("/", httputil.Handle(handlers.Chapter.Create, logger))
						r.Put("/{id}", httputil.Handle(handlers.Chapter.Update, logger))
						r.Delete("/{id}", httputil.Handle(handlers.Chapter.Delete, logger))
					})
				})

				// Lessons
				r.Route("/lessons", func(r chi.Router) {
					r.Get("/{id}", httputil.Handle(handlers.Lesson.GetByID, logger))
					r.Get("/{id}/resources", httputil.Handle(handlers.Lesson.ListResources, logger))

					r.Group(func(r chi.Router) {
						r.Use(mw.RequireRole(model.RoleInstructor))

						r.Post("/", httputil.Handle(handlers.Lesson.Create, logger))
						r.Put("/{id}", httputil.Handle(handlers.Lesson.Update, logger))
						r.Delete("/{id}", httputil.Handle(handlers.Lesson.Delete, logger))
						r.Post("/{id}/resources", httputil.Handle(handlers.Lesson.AddResource, logger))
						r.Put("/resources/{id}", httputil.Handle(handlers.Lesson.UpdateResource, logger))
						r.Delete("/resources/{id}", httputil.Handle(handlers.Lesson.RemoveResource, logger))
					})
				})

				// Bundles
				r.Route("/bundles", func(r chi.Router) {
					r.Use(newRateLimiter(60, time.Minute))

					r.Get("/", httputil.Handle(handlers.Bundle.List, logger))
					r.Get("/{id}", httputil.Handle(handlers.Bundle.GetByID, logger))

					r.Group(func(r chi.Router) {
						r.Use(mw.RequireRole(model.RoleInstructor))

						r.Post("/", httputil.Handle(handlers.Bundle.Create, logger))
						r.Put("/{id}", httputil.Handle(handlers.Bundle.Update, logger))
						r.Put("/{id}/courses", httputil.Handle(handlers.Bundle.SetCourses, logger))
						r.Delete("/{id}", httputil.Handle(handlers.Bundle.Delete, logger))
					})
				})

				// Enrollments
				r.Route("/enrollments", func(r chi.Router) {
					r.Get("/", httputil.Handle(handlers.Enrollment.List, logger))
					r.Get("/{id}", httputil.Handle(handlers.Enrollment.GetByID, logger))
					r.With(newUserRateLimiter(10, time.Minute)).Post("/", httputil.Handle(handlers.Enrollment.Enroll, logger))
					r.Patch("/{id}/status", httputil.Handle(handlers.Enrollment.UpdateStatus, logger))
					r.Delete("/{id}", httputil.Handle(handlers.Enrollment.Delete, logger))
				})

				// Progress
				r.Route("/progress", func(r chi.Router) {
					r.Get("/{user_id}/lessons/{lesson_id}", httputil.Handle(handlers.Progress.Get, logger))
					r.Get("/{user_id}/courses/{course_id}", httputil.Handle(handlers.Progress.ListByCourse, logger))
					r.Post("/", httputil.Handle(handlers.Progress.Save, logger))
				})

				// Certificates
				r.Route("/certificates", func(r chi.Router) {
					r.Get("/", httputil.Handle(handlers.Certificate.ListByUser, logger))
					r.Get("/{id}", httputil.Handle(handlers.Certificate.GetByID, logger))
				})

				// Consents
				r.Route("/consents", func(r chi.Router) {
					r.Get("/", httputil.Handle(handlers.Consent.ListByUser, logger))
					r.Post("/", httputil.Handle(handlers.Consent.Create, logger))
					r.Get("/{id}", httputil.Handle(handlers.Consent.GetByID, logger))
				})

				// Subscriptions
				r.Route("/subscriptions", func(r chi.Router) {
					r.Get("/", httputil.Handle(handlers.Subscriptions.List, logger))
					r.Get("/active", httputil.Handle(handlers.Subscriptions.GetActive, logger))
					r.Post("/", httputil.Handle(handlers.Subscriptions.Subscribe, logger))
					r.Delete("/{id}", httputil.Handle(handlers.Subscriptions.Cancel, logger))
				})

				// Purchases
				r.Route("/purchases", func(r chi.Router) {
					r.Get("/", httputil.Handle(handlers.Purchase.List, logger))
					r.Get("/{id}", httputil.Handle(handlers.Purchase.GetByID, logger))
					r.With(newUserRateLimiter(10, time.Minute)).Post("/courses", httputil.Handle(handlers.Purchase.BuyCourse, logger))
					r.With(newUserRateLimiter(10, time.Minute)).Post("/bundles", httputil.Handle(handlers.Purchase.BuyBundle, logger))
					r.With(newUserRateLimiter(5, time.Minute)).Post("/{id}/refund", httputil.Handle(handlers.Purchase.Refund, logger))
				})
			})
		})
	})

	return r
}
