package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	//"time"
	//"fmt"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/device"
)

var counter int64
var tokens = make(chan struct{}, 6)

func main() {
	ctx := context.Background()
	options := []chromedp.ExecAllocatorOption{
		//chromedp.Flag("disable-sync", false),
		// chromedp.Flag("headless", false),
		//chromedp.WindowSize(3000, 3000),
		//chromedp.Flag("hide-scrollbars", false),
		//chromedp.Flag("mute-audio", false),
		//	chromedp.UserAgent("Mozilla/5.0 (Windows NT 6.3; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36"),
	}

	options = append(chromedp.DefaultExecAllocatorOptions[:], options...)

	c, cc := chromedp.NewExecAllocator(ctx, options...)
	defer cc()

	// create context
	ctx, cancel := chromedp.NewContext(c)
	defer cancel()

	const nRoutineNum = 6
	ctxs := make([]context.Context, 0, 20)
	cancels := make([]context.CancelFunc, 0, 20)
	skus := make([]string, 0, 1000)
	sku := make([]string, 0, 1000)
	var wg sync.WaitGroup
	var nPreCount int
	var strDataSku string
	var nodes []*cdp.Node
	err := chromedp.Run(ctx,
		chromedp.Emulate(device.IPhoneXRlandscape),
		chromedp.Navigate("https://shop.m.jd.com/?shopId=675624"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			chromedp.Click("#shop > div.site_fix_bottom > div:nth-child(2) > span").Do(ctx)
			chromedp.Nodes("#shop > div.site_content.all_product_section > div.module_content > div > div > div > div > div > div > a", &nodes).Do(ctx)
			nPreCount = len(nodes)
			chromedp.Nodes("#shop > div.site_content.all_product_section > div.module_content > div > div > div > div > div", &nodes).Do(ctx)
			for _, v := range nodes {
				strDataSku = v.AttributeValue("data-sku")
				skus = append(skus, strDataSku)
			}
			for i := 0; i < nRoutineNum; i++ {
				ctxNew, cancel := chromedp.NewContext(ctx)
				ctxs = append(ctxs, ctxNew)
				cancels = append(cancels, cancel)
			}
			runtime.Evaluate("window.scrollTo(0,document.body.scrollHeight);").Do(ctx)
			workForVisit(ctxs, skus, nRoutineNum, &wg)
			wg.Wait()
			return nil
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			for _, v := range cancels {
				defer v()
			}
			for {
				chromedp.Nodes("#shop > div.site_content.all_product_section > div.module_content > div > div > div > div > div > div > a", &nodes).Do(ctx)
				nCurCount := len(nodes)
				if nPreCount == nCurCount {
					fmt.Println("当前退出时的项目数量:", nCurCount)
					break
				}
				sku = sku[:0]
				chromedp.Nodes("#shop > div.site_content.all_product_section > div.module_content > div > div > div > div > div", &nodes).Do(ctx)
				for _, v := range nodes {
					sku = append(sku, v.AttributeValue("data-sku"))
				}
				runtime.Evaluate("window.scrollTo(0,document.body.scrollHeight);").Do(ctx)
				workForVisit(ctxs, sku[nPreCount:], nRoutineNum, &wg)
				wg.Wait()
				nPreCount = nCurCount
				fmt.Println("遍历的项目数量:", nCurCount)
			}
			return nil
		}),
	)
	if err != nil {
		log.Fatal(err)
		return
	}
}
func workForVisit(ctxs []context.Context, skus []string, nRoutineNum int, wg *sync.WaitGroup) {
	nCount := len(skus)
	nNumEvery := nCount / nRoutineNum
	nNumLast := nCount % nRoutineNum
	var nNumStep int
	var nIndexCtx int
	var nIndexSku int
	for i := 0; i < nRoutineNum; i++ {
		wg.Add(1)
		if 0 < nNumLast {
			nNumStep = nNumEvery + 1
			nNumLast--
		} else {
			nNumStep = nNumEvery
		}
		// fmt.Println(ctxs[nIndexCtx])
		go visitProductAll(ctxs[nIndexCtx], skus[nIndexSku:nIndexSku+nNumStep], wg)
		nIndexSku += nNumStep
		nIndexCtx++
	}
}
func visitProductAll(ctx context.Context, skus []string, wg *sync.WaitGroup) {
	var strPrice string
	var strProductName string
	var strSpecPrice string
	var strSpecOldPrice string
	defer wg.Done()
	for _, sku := range skus {
		url := fmt.Sprintf("https://item.m.jd.com/product/%s.html", sku)
		err := chromedp.Run(ctx,
			chromedp.Emulate(device.IPhoneXRlandscape),
			chromedp.Navigate(url),
			chromedp.ActionFunc(func(ctx context.Context) error {
				chromedp.Text("#itemName", &strProductName, chromedp.AtLeast(0), chromedp.ByID).Do(ctx)
				chromedp.Text("#priceSale", &strPrice, chromedp.AtLeast(0), chromedp.ByID).Do(ctx)
				if 0 == len(strPrice) {
					chromedp.Text("#specPrice", &strSpecPrice, chromedp.AtLeast(0), chromedp.ByID).Do(ctx)
					chromedp.Text("#specOldPrice", &strSpecOldPrice, chromedp.AtLeast(0), chromedp.ByID).Do(ctx)
				}
				return nil
			}))
		if err != nil {
			log.Fatal(err)
			return
		}
		if 0 == len(strPrice) {
			fmt.Printf("第%d个商品: sku:%s 网址:%s 商品:%s 秒杀价:%s (原价:%s)\n", atomic.AddInt64(&counter, 1), sku, url, strProductName, strSpecPrice, strSpecOldPrice)
		} else {
			fmt.Printf("第%d个商品: sku:%s 网址:%s 商品:%s 价格:%s\n", atomic.AddInt64(&counter, 1), sku, url, strProductName, strPrice)
		}

	}
}
