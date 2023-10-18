package info

import (
	"context"
	"errors"
	"fmt"
	"robot/parametrs"
	"strconv"

	"github.com/adshao/go-binance/v2/futures"
)

func GetInfo(symbol string) float64 {
	// Создание нового клиента Binance Futures API
	client := futures.NewClient(parametrs.ApiKey, parametrs.SecretKey)

	// Создание контекста
	ctx := context.Background()

	// Получение информации о счете
	accountInfo, err := client.NewGetPositionRiskService().Symbol(symbol).Do(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Println("Запрос отменен:", err)
		} else if errors.Is(err, context.DeadlineExceeded) {
			fmt.Println("Запрос превысил время ожидания:", err)
		} else {
			fmt.Println("Ошибка при получении информации о счете:", err)
		}
	}

	// Поиск позиции для указанного символа
	var entryPrice float64
	for _, position := range accountInfo {
		if position.Symbol == symbol {
			entryPrice, err = strconv.ParseFloat(position.EntryPrice, 64)
			if err != nil {
				fmt.Println("Ошибка при преобразовании средней цены входа:", err)
			}
			break
		}
	}

	return entryPrice
}
