package service

import (
	"context"
	"time"

	"github.com/aliexe/ms-priceFetcher/pkg/types"
	"github.com/sirupsen/logrus"
)

type LoggingService struct {
	next PriceService
}

func (s LoggingService) FetchPrice(ctx context.Context, ticker string) (price float64, err error) {
	defer func(begin time.Time) {
		logrus.WithFields(logrus.Fields{
			"requestID": ctx.Value("requestID"),
			"took":      time.Since(begin),
			"err":       err,
			"price":     price,
		}).Info("fetch price")
	}(time.Now())
	return s.next.FetchPrice(ctx, ticker)
}

func (s LoggingService) FetchPrices(ctx context.Context, tickers []string) (prices map[string]float64, err error) {
	defer func(begin time.Time) {
		logrus.WithFields(logrus.Fields{
			"requestID": ctx.Value("requestID"),
			"tickers":   tickers,
			"count":     len(prices),
			"took":      time.Since(begin),
			"err":       err,
		}).Info("fetch prices")
	}(time.Now())
	return s.next.FetchPrices(ctx, tickers)
}

func (s LoggingService) FetchPriceHistory(ctx context.Context, ticker, fromDate, toDate string) (history []types.HistoricalPricePoint, err error) {
	defer func(begin time.Time) {
		logrus.WithFields(logrus.Fields{
			"requestID": ctx.Value("requestID"),
			"ticker":    ticker,
			"from":      fromDate,
			"to":        toDate,
			"count":     len(history),
			"took":      time.Since(begin),
			"err":       err,
		}).Info("fetch price history")
	}(time.Now())
	return s.next.FetchPriceHistory(ctx, ticker, fromDate, toDate)
}

func NewLoggingService(next PriceService) PriceService {
	return &LoggingService{next: next}
}
