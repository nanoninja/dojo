// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

// Package main is the HTTP API server entry point.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nanoninja/dojo/internal/config"
	"github.com/nanoninja/dojo/internal/handler"
	stripepay "github.com/nanoninja/dojo/internal/payment/stripe"
	"github.com/nanoninja/dojo/internal/platform/cache"
	"github.com/nanoninja/dojo/internal/platform/database"
	"github.com/nanoninja/dojo/internal/platform/mailer"
	"github.com/nanoninja/dojo/internal/platform/security"
	"github.com/nanoninja/dojo/internal/router"
	"github.com/nanoninja/dojo/internal/service"
	"github.com/nanoninja/dojo/internal/store"
)

var version = "0.0.0-dev"

// @title           Go API Dojo
// @version         1.0
// @description     REST API with authentication, roles and encryption.

// @host            localhost:8000
// @BasePath        /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Format: Bearer {token}
func main() {
	// Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("application error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	// ==========================================================================
	// Infrastructure
	// ==========================================================================

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.App.Version = version

	db, err := database.Open(cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close() //nolint:errcheck

	redis, err := cache.Open(cfg.Redis)
	if err != nil {
		return err
	}
	defer redis.Close() //nolint:errcheck

	cipher, err := security.NewAESCipher(cfg.App.EncryptionKey)
	if err != nil {
		return fmt.Errorf("cipher: %w", err)
	}

	// ==========================================================================
	// Stores
	// ==========================================================================

	// Auth
	userStore := store.NewUserStore(db, cipher)
	authStore := store.NewAuthStore(db)
	refreshTokenStore := store.NewRefreshTokenStore(db)
	loginAuditStore := store.NewLoginAuditStore(db, cipher)

	// Catalog
	courseStore := store.NewCourseStore(db)
	coursesCategoriesStore := store.NewCoursesCategoriesStore(db)
	coursesTagsStore := store.NewCoursesTagsStore(db)
	categoryStore := store.NewCategoryStore(db)
	tagStore := store.NewTagStore(db)
	chapterStore := store.NewChapterStore(db)
	lessonStore := store.NewLessonStore(db)
	lessonResourceStore := store.NewLessonResourceStore(db)

	// Enrollments & Progress
	enrollmentStore := store.NewEnrollmentStore(db)
	progressStore := store.NewLessonProgressStore(db)

	// Business
	bundleStore := store.NewBundleStore(db)
	bundleCourseStore := store.NewBundleCourseStore(db)
	reviewStore := store.NewReviewStore(db)
	certificateStore := store.NewCertificateStore(db)
	consentStore := store.NewConsentStore(db, cipher)
	subscriptionStore := store.NewSubscriptionStore(db)
	purchaseStore := store.NewPurchaseStore(db)

	// ==========================================================================
	// Services
	// ==========================================================================

	// Auth mailer
	baseAuthMailer := service.NewAuthMailer(mailer.NewSMTP(mailer.SMTPConfig{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		Username: cfg.SMTP.Username,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
	}), mailer.ParseTemplates())

	authMailer := service.NewResilientAuthMailer(baseAuthMailer, service.MailDispatchConfig{
		Enabled:        cfg.MailDispatch.Enabled,
		Timeout:        time.Duration(cfg.MailDispatch.TimeoutMS) * time.Millisecond,
		RetryAttempts:  cfg.MailDispatch.RetryAttempts,
		RetryBaseDelay: time.Duration(cfg.MailDispatch.RetryBaseDelay) * time.Millisecond,
	})

	// Auth
	userService := service.NewUserService(userStore, loginAuditStore)
	authService := service.NewAuthService(userStore, authStore, refreshTokenStore,
		loginAuditStore,
		authMailer,
		cfg.JWT,
		logger,
	)

	// Catalog
	courseService := service.NewCourseService(db, courseStore, coursesCategoriesStore, coursesTagsStore)
	categoryService := service.NewCategoryService(categoryStore)
	tagService := service.NewTagService(tagStore)
	chapterService := service.NewChapterService(chapterStore)
	lessonService := service.NewLessonService(lessonStore, lessonResourceStore)

	// Enrollments & Progress
	enrollmentService := service.NewEnrollmentService(enrollmentStore)
	progressService := service.NewLessonProgressService(db, progressStore, enrollmentStore, courseStore)

	// Business
	bundleService := service.NewBundleService(db, bundleStore, bundleCourseStore)
	reviewService := service.NewReviewService(db, reviewStore)
	certificateService := service.NewCertificateService(certificateStore)
	consentService := service.NewConsentService(consentStore)
	subscriptionService := service.NewSubscriptionService(subscriptionStore)
	stripeClient := stripepay.New(stripepay.Config{
		SecretKey:     cfg.Stripe.SecretKey,
		WebhookSecret: cfg.Stripe.WebhookSecret,
		SuccessURL:    cfg.Stripe.SuccessURL,
		CancelURL:     cfg.Stripe.CancelURL,
	})

	purchaseService := service.NewPurchaseService(db, stripeClient, purchaseStore, enrollmentStore, bundleCourseStore)
	accessService := service.NewAccessService(subscriptionStore, enrollmentStore)

	// ==========================================================================
	// Background jobs
	// ==========================================================================

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runAuditPurge(ctx, cfg.AuditPurge, loginAuditStore, db, logger)

	// ==========================================================================
	// Handlers
	// ==========================================================================

	var wg sync.WaitGroup

	handlers := &router.Handlers{
		// Auth
		Auth: handler.NewAuthHandler(authService, userService, cfg.AuthTransport, cfg.JWT, logger, &wg),
		User: handler.NewUserHandler(userService),

		// Catalog
		Course:   handler.NewCourseHandler(courseService, service.NewCourseOwnership(db)),
		Category: handler.NewCategoryHandler(categoryService),
		Tag:      handler.NewTagHandler(tagService),
		Chapter:  handler.NewChapterHandler(chapterService, service.NewChapterOwnership(db), service.NewCourseOwnership(db), accessService),
		Lesson:   handler.NewLessonHandler(lessonService, chapterService, service.NewLessonOwnership(db), service.NewChapterOwnership(db), accessService),

		// Enrollments & Progress
		Enrollment: handler.NewEnrollmentHandler(enrollmentService),
		Progress:   handler.NewProgressHandler(progressService),

		// Business
		Bundle:        handler.NewBundleHandler(bundleService, service.NewBundleOwnership(db)),
		Review:        handler.NewReviewHandler(reviewService),
		Certificate:   handler.NewCertificateHandler(certificateService),
		Consent:       handler.NewConsentHandler(consentService),
		Subscriptions: handler.NewSubscriptionHandler(subscriptionService),
		Purchase:      handler.NewPurchaseHandler(purchaseService),
		Webhook:       handler.NewWebhookHandler(stripeClient, purchaseService),
	}

	// ==========================================================================
	// Server
	// ==========================================================================

	srv := http.Server{
		Addr:         cfg.App.Host + ":" + cfg.App.Port,
		Handler:      router.New(handlers, cfg, logger, db, redis),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine so it does not block the signal listener.
	errCh := make(chan error, 1)

	go func() {
		logger.Info("starting server", "env", cfg.App.Env, "port", cfg.App.Port)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// Block until a signal or a server error.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case <-quit:
		// intentionally empty, proceed to graceful shutdown
	}

	// Allow up to 30 seconds for in-flight requests to complete.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	// Wait for in-flight async operations (email sends) to complete
	wg.Wait()

	logger.Info("server stopped")
	return nil
}
