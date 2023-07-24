package main

import (
	"encoding/json"
	"log"
	"net/http"
	"robot/createOrder"
	"robot/parametrs"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
)

// Структура для получения информации о фьючерсах
type Info struct {
	Symbols []struct {
		Symbol       string `json:"symbol"`
		ContractType string `json:"contractType"`
	} `json:"symbols"`
}

// Функция для получения списка всех фьючерсов с помощью API Binance
func getFuturesInfo() ([]string, error) {
	resp, err := http.Get("https://fapi.binance.com/fapi/v1/exchangeInfo")
	if err != nil {
		panic("Failed to get futures exchange info")
	}
	defer resp.Body.Close()

	var exchangeInfo Info
	err = json.NewDecoder(resp.Body).Decode(&exchangeInfo)
	if err != nil {
		panic("Failed to parse futures exchange info")
	}

	symbols := []string{}
	for _, symbol := range exchangeInfo.Symbols {
		if symbol.ContractType != "PERPETUAL" {
			continue
		}
		symbols = append(symbols, symbol.Symbol)
	}
	// symbols = []string{"ALPHAUSDT", "XEMUSDT", "CFXUSDT"} //  фьючерсы для теста
	return symbols, nil
}

// Функция для подключения к WebSocket API Binance и получения свечных данных фьючерса
func candlestickData(symbol string) {
	endpoint := "wss://fstream.binance.com/ws/" + strings.ToLower(symbol) + "@kline_1m"
	ws, _, err := websocket.DefaultDialer.Dial(endpoint, nil)
	if err != nil {
		log.Printf("Ошибка WebSocket для символа %s: %s", symbol, err)
		return
	}
	defer ws.Close()

	sliceAmplTmp := []float64{}
	sliceAmpl := []float64{}
	avgAmpl := 1.0
	position := false
	count := false
	sides := ""
	take_profit := 0.0
	stop_loss := 0.0
	quantity := ""

	for {

		for !position {
			_, candle, err := ws.ReadMessage()
			if err != nil {
				log.Printf("Ошибка чтения свечи для символа %s: %s", symbol, err)
				return
			}

			var klineData struct {
				K struct {
					High   interface{} `json:"h"`
					Low    interface{} `json:"l"`
					Close  interface{} `json:"c"`
					X      interface{} `json:"x"`
					Volume interface{} `json:"q"`
					VolBuy interface{} `json:"Q"`
				} `json:"k"`
			}

			if err = json.Unmarshal(candle, &klineData); err != nil {
				log.Printf("Ошибка расшифровки свечи для символа %s: %s", symbol, err)
				continue
			}

			klineHigh, err := strconv.ParseFloat(klineData.K.High.(string), 64)
			if err != nil {
				log.Printf("Ошибка преобразования значения %v в float64 для символа %s: %s", klineData.K.High, symbol, err)
				continue
			}

			klineLow, err := strconv.ParseFloat(klineData.K.Low.(string), 64)
			if err != nil {
				log.Printf("Ошибка преобразования значения %v в float64 для символа %s: %s", klineData.K.Low, symbol, err)
				continue
			}

			klineClose, err := strconv.ParseFloat(klineData.K.Close.(string), 64)
			if err != nil {
				log.Printf("Ошибка преобразования значения %v в float64 для символа %s: %s", klineData.K.Close, symbol, err)
				continue
			}

			klineVolume, err := strconv.ParseFloat(klineData.K.Volume.(string), 64)
			if err != nil {
				log.Printf("Ошибка преобразования значения %v в float64 для символа %s: %s", klineData.K.Volume, symbol, err)
				continue
			}

			klineVolBuy, err := strconv.ParseFloat(klineData.K.VolBuy.(string), 64)
			if err != nil {
				log.Printf("Ошибка преобразования значения %v в float64 для символа %s: %s", klineData.K.VolBuy, symbol, err)
				continue
			}

			amplitude := (klineHigh - klineLow) / klineLow * 100

			if klineData.K.X == true {
				sliceAmplTmp = append(sliceAmplTmp, float64(amplitude))
				if len(sliceAmplTmp) == parametrs.LenSliceAmplTmp {
					count = true
				}
			}

			if !count {
				continue
			}
			if klineData.K.X == true {
				sliceAmpl = sliceAmplTmp[:parametrs.LenSliceAmplTmp-1]
				sliceAmplTmp = sliceAmplTmp[1:]
				sumAmpl := 0.0
				for _, ampl := range sliceAmpl {
					sumAmpl += ampl
				}
				avgAmpl = sumAmpl / float64(len(sliceAmpl))
			}

			if amplitude > float64(avgAmpl)*parametrs.Koef && avgAmpl >= parametrs.PorogAvgAmpl {
				if (klineVolume - klineVolBuy) == 0.0 {
					continue
				}
				buyDivSell := klineVolBuy / (klineVolume - klineVolBuy)
				otnoshAmpl := amplitude / float64(avgAmpl)
				if buyDivSell > 1.1 && klineClose < klineHigh*(100-parametrs.MinFromAmpl*amplitude)/100 && klineClose > klineHigh*(100-parametrs.MaxFromAmpl*amplitude)/100 && amplitude > parametrs.PorogAmplShort {
					sides = "short"
					quantity = createOrder.SizeLot(parametrs.Dollars, klineClose, parametrs.Koef_stop)
					createOrder.CreateOrder(symbol, sides, quantity)
					position = true
					take_profit = klineClose * (1 - amplitude*parametrs.ForTake)
					stop_loss = klineClose * parametrs.ForStopShort
					log.Printf("short %s \t price: %v \t ampl: %.2f%% \t avg: %.2f%% \t otnosh: %.2f \t buyDivSell: %.2f \t take_profit: %f \t stop_loss: %f", symbol, klineClose, amplitude, avgAmpl, otnoshAmpl, buyDivSell, take_profit, stop_loss)
				}
				if buyDivSell < 0.9 && klineClose > klineLow*(100+parametrs.MinFromAmpl*amplitude)/100 && klineClose < klineLow*(100+parametrs.MaxFromAmpl*amplitude)/100 && amplitude > parametrs.PorogAmplLong {
					sides = "long"
					quantity = createOrder.SizeLot(parametrs.Dollars, klineClose, parametrs.Koef_stop)
					createOrder.CreateOrder(symbol, sides, quantity)
					position = true
					take_profit = klineClose * (1 + amplitude*parametrs.ForTake)
					stop_loss = klineClose * parametrs.ForStopLong
					log.Printf("long %s \t price: %v \t ampl: %.2f%% \t avg: %.2f%% \t otnosh: %.2f \t buyDivSell: %.2f \t take_profit: %f \t stop_loss: %f", symbol, klineClose, amplitude, avgAmpl, otnoshAmpl, buyDivSell, take_profit, stop_loss)
				}
			}
		}

		// ---------------   Сделка открыта   --------------- \\

		for position {
			_, candle, err := ws.ReadMessage()
			if err != nil {
				log.Printf("Ошибка чтения свечи для символа %s: %s", symbol, err)
				return
			}

			var klineData struct {
				K struct {
					High   interface{} `json:"h"`
					Low    interface{} `json:"l"`
					Close  interface{} `json:"c"`
					X      interface{} `json:"x"`
					Volume interface{} `json:"q"`
					VolBuy interface{} `json:"Q"`
				} `json:"k"`
			}

			if err = json.Unmarshal(candle, &klineData); err != nil {
				log.Printf("Ошибка расшифровки свечи для символа %s: %s", symbol, err)
				continue
			}

			klineHigh, err := strconv.ParseFloat(klineData.K.High.(string), 64)
			if err != nil {
				log.Printf("Ошибка преобразования значения %v в float64 для символа %s: %s", klineData.K.High, symbol, err)
				continue
			}

			klineLow, err := strconv.ParseFloat(klineData.K.Low.(string), 64)
			if err != nil {
				log.Printf("Ошибка преобразования значения %v в float64 для символа %s: %s", klineData.K.Low, symbol, err)
				continue
			}

			klineClose, err := strconv.ParseFloat(klineData.K.Close.(string), 64)
			if err != nil {
				log.Printf("Ошибка преобразования значения %v в float64 для символа %s: %s", klineData.K.Close, symbol, err)
				continue
			}

			amplitude := (klineHigh - klineLow) / klineLow * 100

			if klineData.K.X == true {
				sliceAmplTmp = append(sliceAmplTmp, float64(amplitude))
				if len(sliceAmplTmp) == parametrs.LenSliceAmplTmp {
				}
			}

			if klineData.K.X == true {
				sliceAmpl = sliceAmplTmp[:parametrs.LenSliceAmplTmp-1]
				sliceAmplTmp = sliceAmplTmp[1:]
				sumAmpl := 0.0
				for _, ampl := range sliceAmpl {
					sumAmpl += ampl
				}
				avgAmpl = sumAmpl / float64(len(sliceAmpl))
			}

			if sides == "long" {
				sides = "short"
				if klineClose >= take_profit {
					createOrder.CreateOrder(symbol, sides, quantity)
					log.Printf("take_profit %s \t price: %v \t koef_stop: %v", symbol, klineClose, parametrs.Koef_stop)
					position = false
					parametrs.Koef_stop = parametrs.Koef_stop_default
				}
				if klineClose <= stop_loss {
					createOrder.CreateOrder(symbol, sides, quantity)
					log.Printf("stop_loss %s \t price: %v \t koef_stop: %v", symbol, klineClose, parametrs.Koef_stop)
					position = false
					parametrs.Koef_stop *= 2
				}
				sides = "long"
			}

			if sides == "short" {
				sides = "long"
				if klineClose <= take_profit {
					createOrder.CreateOrder(symbol, sides, quantity)
					log.Printf("take_profit %s \t price: %v \t koef_stop: %v", symbol, klineClose, parametrs.Koef_stop)
					position = false
					parametrs.Koef_stop = parametrs.Koef_stop_default
				}
				if klineClose >= stop_loss {
					createOrder.CreateOrder(symbol, sides, quantity)
					log.Printf("stop_loss %s \t price: %v \t koef_stop: %v", symbol, klineClose, parametrs.Koef_stop)
					position = false
					parametrs.Koef_stop *= 2
				}
				sides = "short"
			}
		}
	}
}

func main() {
	futuresInfo, err := getFuturesInfo()
	if err != nil {
		log.Fatalf("Ошибка получения списка фьючерсов: %s", err)
	}

	for _, symbol := range futuresInfo {

		go candlestickData(symbol)
	}
	log.Println("Start")

	select {} // Ожидание бесконечного потока
}
