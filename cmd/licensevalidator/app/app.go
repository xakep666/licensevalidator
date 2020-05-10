package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"regexp"
	"time"

	"github.com/Masterminds/semver/v3"
	gh "github.com/google/go-github/v18/github"
	"github.com/mediocregopher/radix/v3"
	goprom "github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/exporters/metric/prometheus"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	"go.opentelemetry.io/otel/exporters/trace/zipkin"
	"go.opentelemetry.io/otel/plugin/othttp"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"github.com/xakep666/licensevalidator/pkg/athens"
	"github.com/xakep666/licensevalidator/pkg/cache"
	"github.com/xakep666/licensevalidator/pkg/github"
	"github.com/xakep666/licensevalidator/pkg/golang"
	"github.com/xakep666/licensevalidator/pkg/gopkg"
	"github.com/xakep666/licensevalidator/pkg/goproxy"
	"github.com/xakep666/licensevalidator/pkg/observ"
	"github.com/xakep666/licensevalidator/pkg/override"
	"github.com/xakep666/licensevalidator/pkg/spdx"
	"github.com/xakep666/licensevalidator/pkg/validation"
)

type App struct {
	logger      *zap.Logger
	server      *http.Server
	tracerFlush func()
}

func NewApp(cfg Config) (*App, error) {
	var logger *zap.Logger
	if cfg.Debug {
		logger, _ = zap.NewDevelopment()
	} else {
		logger, _ = zap.NewProduction()
	}

	logger.Info("Running with config", zap.Reflect("config", cfg))

	tracer, tracerFlush, err := setupTracer(&cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("tracer setup failed: %w", err)
	}

	pushController, metricHandler, err := setupPrometheus(logger)
	if err != nil {
		return nil, fmt.Errorf("prometheus init failed: %w", err)
	}

	meter := pushController.Meter("")

	translator, err := translator(logger, &cfg)
	if err != nil {
		return nil, fmt.Errorf("translator init failed: %w", err)
	}

	c, err := setupCache(&cfg, cache.Direct{
		LicenseResolver: &observ.LicenseResolver{
			LicenseResolver: &validation.ChainedLicenseResolver{
				LicenseResolvers: []validation.LicenseResolver{
					githubClient(logger, &cfg, tracer, meter),
					goproxyClient(logger, &cfg, tracer, meter),
				},
			},
			Meter: meter,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("setup cache failed: %w", err)
	}

	validator, err := validator(logger, &cfg, translator, c)
	if err != nil {
		return nil, fmt.Errorf("validator init failed: %w", err)
	}

	logger.Info("Trying to resolve goproxy addresses", zap.String("goproxy", string(cfg.GoProxy.BaseURL)))

	goproxyAddrs, err := goproxyAddrs(&cfg)
	if err != nil {
		return nil, fmt.Errorf("get goproxy addrs failed: %w", err)
	}

	logger.Info("Found forbidden admission request sources", zap.Strings("sources", goproxyAddrs))

	mux := http.NewServeMux()
	observMiddleware := observ.Middleware(logger, pushController.Meter("http_requests"))
	mux.Handle("/athens/admission",
		othttp.NewHandler(
			observMiddleware(
				athens.AdmissionHandler(
					&athens.InternalValidator{Validator: validator},
					goproxyAddrs...,
				),
			),
			"athens admission hook",
			othttp.WithTracer(tracer),
		),
	)
	mux.HandleFunc("/metrics", metricHandler)
	addPprofHandlers(&cfg, mux)

	return &App{
		logger: logger,
		server: &http.Server{
			Addr:    cfg.Server.ListenAddr,
			Handler: mux,
			ErrorLog: func() *log.Logger {
				l, _ := zap.NewStdLogAt(logger, zap.ErrorLevel)
				return l
			}(),
		},
		tracerFlush: tracerFlush,
	}, nil
}

func (a *App) Run() error {
	a.logger.Info("Serving HTTP Requests", zap.String("listen_addr", a.server.Addr))
	err := a.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}

func (a *App) Stop(ctx context.Context) error {
	a.logger.Info("Stopping")
	defer a.tracerFlush()
	return a.server.Shutdown(ctx)
}

func setupRedis(cfg *Config) (radix.Client, error) {
	addrs := cfg.Cache.Redis.Addrs
	if len(addrs) == 0 {
		return nil, fmt.Errorf("redis addres(es) required for using redis as cache")
	}

	var dialOpts []radix.DialOpt
	if cfg.Cache.Redis.DB > 0 {
		dialOpts = append(dialOpts, radix.DialSelectDB(cfg.Cache.Redis.DB))
	}
	if cfg.Cache.Redis.Password != "" {
		dialOpts = append(dialOpts, radix.DialAuthPass(cfg.Cache.Redis.Password))
	}
	if cfg.Cache.Redis.ConnectTimeout > 0 {
		dialOpts = append(dialOpts, radix.DialConnectTimeout(cfg.Cache.Redis.ConnectTimeout))
	}
	if cfg.Cache.Redis.ReadTimeout > 0 {
		dialOpts = append(dialOpts, radix.DialReadTimeout(cfg.Cache.Redis.ReadTimeout))
	}
	if cfg.Cache.Redis.WriteTimeout > 0 {
		dialOpts = append(dialOpts, radix.DialWriteTimeout(cfg.Cache.Redis.WriteTimeout))
	}

	customConnFunc := func(network, addr string) (radix.Conn, error) {
		return radix.Dial(network, addr, dialOpts...)
	}

	poolSize := 10
	if cfg.Cache.Redis.PoolSize > 0 {
		poolSize = cfg.Cache.Redis.PoolSize
	}

	if len(addrs) == 1 {
		return radix.NewPool("tcp", cfg.Cache.Redis.Addrs[0], poolSize, radix.PoolConnFunc(customConnFunc))
	} else {
		return radix.NewCluster(cfg.Cache.Redis.Addrs, radix.ClusterPoolFunc(func(network, addr string) (radix.Client, error) {
			return radix.NewPool(network, addr, poolSize, radix.PoolConnFunc(customConnFunc))
		}))
	}
}

func setupCache(cfg *Config, cacher cache.Cacher) (cache.Cacher, error) {
	if cfg.Cache == nil {
		return cacher, nil
	}

	switch cfg.Cache.Type {
	case CacheTypeMemory:
		return &cache.MemoryCache{
			Backed: cacher,
		}, nil
	case CacheTypeMemLRU:
		return cache.NewMemLRU(cacher, cfg.Cache.SizeItems)
	case CacheTypeRedis:
		redisClient, err := setupRedis(cfg)
		if err != nil {
			return nil, fmt.Errorf("redis client setup failed: %w", err)
		}
		return &cache.RedisCache{
			Backed: cacher,
			Client: redisClient,
			TTL:    cfg.Cache.Redis.TTL,
		}, nil
	default:
		return nil, fmt.Errorf("invalid cache type: %s", cfg.Cache.Type)
	}
}

func githubClient(log *zap.Logger, cfg *Config, tracer trace.Tracer, meter metric.Meter) *github.Client {
	httpClient := &http.Client{}

	if cfg.Github.AccessToken != "" {
		httpClient = oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: string(cfg.Github.AccessToken),
		}))
	}

	httpClient.Transport = &observ.TraceTransport{
		RoundTripper: httpClient.Transport,
		ServiceName:  "github",
		Tracer:       tracer,
		Meter:        meter,
	}

	return github.NewClient(log, github.ClientParams{
		Client:                      gh.NewClient(httpClient),
		FallbackConfidenceThreshold: cfg.Validation.ConfidenceThreshold,
	})
}

func goproxyClient(log *zap.Logger, cfg *Config, tracer trace.Tracer, meter metric.Meter) *goproxy.Client {
	if cfg.GoProxy.BaseURL == "" {
		cfg.GoProxy.BaseURL = "https://proxy.golang.org"
	}
	return goproxy.NewClient(log, goproxy.ClientParams{
		HTTPClient: &http.Client{
			Transport: &observ.TraceTransport{
				ServiceName: "goproxy",
				Tracer:      tracer,
				Meter:       meter,
			},
		},
		BaseURL:             string(cfg.GoProxy.BaseURL),
		ConfidenceThreshold: cfg.Validation.ConfidenceThreshold,
	})
}

func goproxyAddrs(cfg *Config) ([]string, error) {
	u, err := url.Parse(string(cfg.GoProxy.BaseURL))
	if err != nil {
		return nil, fmt.Errorf("goproxy url parse failed: %w", err)
	}

	ips, err := (&net.Resolver{
		PreferGo: true,
	}).LookupIPAddr(context.Background(), u.Hostname())
	if err != nil {
		return nil, fmt.Errorf("goproxy addresses lookup failed: %w", err)
	}

	var addrs []string
	for _, v := range ips {
		addrs = append(addrs, v.String())
	}

	return append(addrs, u.Hostname(), u.Host), nil
}

func translator(log *zap.Logger, cfg *Config) (*validation.ChainedTranslator, error) {
	var overrides []override.TranslateOverride

	for _, item := range cfg.PathOverrides {
		m, err := regexp.Compile(item.Match)
		if err != nil {
			return nil, fmt.Errorf("invalid match %s: %w", item.Match, err)
		}

		overrides = append(overrides, override.TranslateOverride{
			Match:   m,
			Replace: item.Replace,
		})
	}

	return &validation.ChainedTranslator{
		Translators: []validation.Translator{
			override.NewTranslator(log, overrides),
			golang.Translator{},
			gopkg.Translator{},
		},
	}, nil
}

func validator(log *zap.Logger, cfg *Config, translator validation.Translator, resolver validation.LicenseResolver) (*validation.NotifyingValidator, error) {
	var unknownLicenseAction validation.UnknownLicenseAction

	switch cfg.Validation.UnknownLicenseAction {
	case UnknownLicenseAllow:
		unknownLicenseAction = validation.UnknownLicenseAllow
	case UnknownLicenseWarn:
		// TODO
		// unknownLicenseAction = validation.UnknownLicenseWarn
		return nil, fmt.Errorf("warning about unknown license currently not supported")
	case UnknownLicenseDeny:
		unknownLicenseAction = validation.UnknownLicenseDeny
	default:
		return nil, fmt.Errorf("unexpected unknown license action %s", cfg.Validation.UnknownLicenseAction)
	}

	var (
		ruleSet validation.RuleSet
		err     error
	)

	ruleSet.WhitelistedModules, err = parseModuleMatchers(cfg.Validation.RuleSet.WhitelistedModules)
	if err != nil {
		return nil, fmt.Errorf("whitelisted modules parse failed: %w", err)
	}

	ruleSet.BlacklistedModules, err = parseModuleMatchers(cfg.Validation.RuleSet.BlacklistedModules)
	if err != nil {
		return nil, fmt.Errorf("blacklisted modules parse failed: %w", err)
	}

	ruleSet.AllowedLicenses, err = parseLicenses(cfg.Validation.RuleSet.AllowedLicenses)
	if err != nil {
		return nil, fmt.Errorf("allowed licenses parse failed: %w", err)
	}

	ruleSet.DeniedLicenses, err = parseLicenses(cfg.Validation.RuleSet.DeniedLicenses)
	if err != nil {
		return nil, fmt.Errorf("denied licenses parse failed: %w", err)
	}

	return validation.NewNotifyingValidator(
		log, validation.NotifyingValidatorParams{
			Validator: validation.NewRuleSetValidator(log, validation.RuleSetValidatorParams{
				Translator:      translator,
				LicenseResolver: resolver,
				RuleSet:         ruleSet,
			}),
			UnknownLicenseAction: unknownLicenseAction,
		}), nil
}

func parseModuleMatchers(ms []ModuleMatcher) ([]validation.ModuleMatcher, error) {
	ret := make([]validation.ModuleMatcher, 0, len(ms))
	for _, item := range ms {
		if item.Name == "" {
			return nil, fmt.Errorf("module name matcher can't have empty name")
		}
		name, err := regexp.Compile(item.Name)
		if err != nil {
			return nil, fmt.Errorf("invalid module name matcher regexp %s: %w", item.Name, err)
		}

		var constraint *semver.Constraints
		if item.VersionConstraint != "" {
			constraint, err = semver.NewConstraint(item.VersionConstraint)
			if err != nil {
				return nil, fmt.Errorf("invalid constraint for module %s (%s): %w", item.Name, item.VersionConstraint, err)
			}
		}

		ret = append(ret, validation.ModuleMatcher{
			Name:    name,
			Version: constraint,
		})
	}

	return ret, nil
}

func parseLicenses(ls []License) ([]validation.License, error) {
	ret := make([]validation.License, 0, len(ls))
	for _, item := range ls {
		var license validation.License

		if item.SPDXID != "" {
			lic, ok := spdx.LicenseByID(item.SPDXID)
			if !ok {
				return nil, fmt.Errorf("license %s not found in SPDX", item.SPDXID)
			}

			license.SPDXID = item.SPDXID
			license.Name = lic.Name
		} else {
			license.Name = item.Name
		}

		ret = append(ret, license)
	}

	return ret, nil
}

func addPprofHandlers(cfg *Config, mux *http.ServeMux) {
	if cfg.Server.EnablePprof {
		mux.HandleFunc("/pprof/", pprof.Index)
		mux.HandleFunc("/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/pprof/profile", pprof.Profile)
		mux.HandleFunc("/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/pprof/trace", pprof.Trace)
	}
}

func setupTracer(cfg *Config, logger *zap.Logger) (tracer trace.Tracer, flush func(), err error) {
	if cfg.Trace == nil {
		return trace.NoopTracer{}, func() {}, nil
	}

	sampler := sdktrace.AlwaysSample()
	if cfg.Trace.SampleProbability > 0 {
		sampler = sdktrace.ProbabilitySampler(cfg.Trace.SampleProbability)
	}

	switch cfg.Trace.TracerType {
	case JaegerTracer:
		logger := logger.With(zap.String("component", "jaeger_exporter"))
		jt, flush, err := jaeger.NewExportPipeline(
			jaeger.WithCollectorEndpoint(cfg.Trace.CollectorAddress),
			jaeger.WithProcess(jaeger.Process{
				ServiceName: "licensevalidator",
				Tags: []core.KeyValue{
					key.String("exporter", "jaeger"),
				},
			}),
			jaeger.RegisterAsGlobal(),
			jaeger.WithSDK(&sdktrace.Config{DefaultSampler: sampler}),
			jaeger.WithOnError(func(err error) {
				logger.Error("span upload failed", zap.Error(err))
			}),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("jaeger setup failed: %w", err)
		}

		return jt.Tracer(""), flush, nil
	case ZipkinTracer:
		zexp, err := zipkin.NewExporter(
			cfg.Trace.CollectorAddress,
			"licensevalidator",
			zipkin.WithLogger(zap.NewStdLog(logger.With(zap.String("component", "zipkin_exporter")))),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("zipkin exporter setup failed: %w", err)
		}

		tp, err := sdktrace.NewProvider(
			sdktrace.WithBatcher(zexp),
			sdktrace.WithResourceAttributes(key.String("exporter", "zipkin")),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("zipkin trace provider setup failed: %w", err)
		}

		return tp.Tracer(""), func() {}, nil
	default:
		return nil, nil, fmt.Errorf("unknown tracer type %s", cfg.Trace.TracerType)
	}
}

func setupPrometheus(logger *zap.Logger) (*push.Controller, http.HandlerFunc, error) {
	logger = logger.With(zap.String("component", "prometheus_push_controller"))

	reg := goprom.NewPedanticRegistry()
	reg.MustRegister(
		goprom.NewGoCollector(),
		goprom.NewBuildInfoCollector(),
	)

	return prometheus.NewExportPipeline(prometheus.Config{
		Registry:                reg,
		DefaultSummaryQuantiles: []float64{0.5, 0.9, 0.99, 1},
		DefaultHistogramBoundaries: []core.Number{
			core.NewFloat64Number(.0001),
			core.NewFloat64Number(.0003),
			core.NewFloat64Number(.0005),
			core.NewFloat64Number(.001),
			core.NewFloat64Number(.015),
			core.NewFloat64Number(.02),
			core.NewFloat64Number(.03),
			core.NewFloat64Number(.05),
			core.NewFloat64Number(.07),
			core.NewFloat64Number(.01),
			core.NewFloat64Number(.15),
			core.NewFloat64Number(.2),
			core.NewFloat64Number(.3),
			core.NewFloat64Number(.4),
			core.NewFloat64Number(.5),
			core.NewFloat64Number(.75),
			core.NewFloat64Number(1),
			core.NewFloat64Number(1.4),
			core.NewFloat64Number(2),
			core.NewFloat64Number(3),
			core.NewFloat64Number(4),
			core.NewFloat64Number(5),
			core.NewFloat64Number(7),
			core.NewFloat64Number(10),
			core.NewFloat64Number(15),
		},
		OnError: func(err error) {
			logger.Error("Push controller error", zap.Error(err))
		},
	},
		10*time.Second,
	)
}
