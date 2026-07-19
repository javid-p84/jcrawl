package scraper

import (
	"context"
	"time"

	"github.com/chromedp/chromedp"
)

type BrowserPool struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func NewBrowserPool() (*BrowserPool, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, _ := chromedp.NewContext(allocCtx)

	bp := &BrowserPool{
		ctx:    ctx,
		cancel: cancel,
	}

	return bp, nil
}

func (bp *BrowserPool) GetContent(url string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(bp.ctx, timeout)
	defer cancel()

	var htmlContent string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second), // Wait for dynamic content to load
		chromedp.OuterHTML("html", &htmlContent),
	)

	if err != nil {
		return "", err
	}

	return htmlContent, nil
}

func (bp *BrowserPool) Close() error {
	bp.cancel()
	return nil
}
