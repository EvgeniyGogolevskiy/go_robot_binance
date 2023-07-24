package createOrder

import (
	"context"
	"math"
	"log"
	"os"
	"robot/parametrs"
	"fmt"

	"github.com/adshao/go-binance/v2/futures"
)

func SizeLot(quantity, price, koef float64) string {
	var lot float64
	var strlot string
   
	lot = quantity / price * koef
   
	if price <= quantity {
	 lot = math.Round(lot)
	} else if quantity < price && price <= quantity*10 {
	 lot = math.Round(lot*10) / 10
	} else if quantity*10 < price && price <= quantity*100 {
	 lot = math.Round(lot*100) / 100
	} else if quantity*100 < price && price <= quantity*1000 {
	 lot = math.Round(lot*1000) / 1000
	} else {
	 lot = math.Round(lot*10000) / 10000
	}

	strlot = fmt.Sprintf("%f", lot)
   
	return strlot
   }

func CreateOrder(symbol, sides, quantity string) {

 // Создание клиента Binance Futures
 client := futures.NewClient(parametrs.ApiKey, parametrs.SecretKey)

 // Создание сервиса создания ордера
 createOrderService := client.NewCreateOrderService()

 // Установка параметров для маркет-ордера
 side := futures.SideTypeBuy

 if sides == "short" {
	side = futures.SideTypeSell
 }

 // Установка параметров ордера в сервисе создания ордера
 createOrderService.Symbol(symbol).Side(side).Quantity(quantity).Type(futures.OrderTypeMarket)

 ctx := context.Background()

 // Открытие маркет-ордера
 _, err := createOrderService.Do(ctx)
 if err != nil {
  log.Println("Ошибка при создании маркет-ордера:", err)
  os.Exit(1)
 }
}