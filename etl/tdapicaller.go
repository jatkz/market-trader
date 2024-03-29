package etl

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/avast/retry-go"
	"github.com/jaredtokuz/market-trader/shared"
	"github.com/jaredtokuz/market-trader/token"
)

type tdapiconfig struct {
	mongo  *MongoController
	apikey string
	token  token.AccessTokenService
}

type TDApiService interface {
	Call(etlConfig EtlConfig) (ApiCallSuccess, error)                                       /* Makes a request */
	AddAuth(req *http.Request)                                                              /* helper */
	AddApiKey(req *url.Values)                                                              /* helper */
	InsertResponse(etlConfig EtlConfig, resp *http.Response, decodedBody interface{}) error /* log api response */
}

func NewTDApiService(
	mongo *MongoController,
	apikey string,
	token token.AccessTokenService,
) TDApiService {
	return &tdapiconfig{mongo: mongo, apikey: apikey, token: token}
}

func (i *tdapiconfig) Call(etlConfig EtlConfig) (ApiCallSuccess, error) {
	// retryClient := retryablehttp.NewClient() //.Backoff(time.Duration(2)*time.Second, time.Duration(5)*time.Second, 5, resp) //LinearJitterBackoff(time.Duration(1)*time.Second, time.Duration(3)*time.Second, 5, resp)

	// retryClient.RetryMax = 4
	// retryClient.RetryWaitMin = time.Duration(1) * time.Second
	// retryClient.RetryWaitMin = time.Duration(3) * time.Second

	// client := retryClient.StandardClient() // convert to *http.Client
	client := &http.Client{}

	var (
		body map[string]interface{}
	)
	err := retry.Do(
		func() error {
			var (
				req *http.Request
				err error
			)
			// Dynamically set url/method
			switch etlConfig.Work {
			case Macros:
				req, err = http.NewRequest("GET", InstrumentsUrl, nil)
			case Medium, Short, Signals:
				req, err = http.NewRequest("GET", PriceHistoryUrl(etlConfig.Symbol), nil)
			}
			query := req.URL.Query()
			i.AddAuth(req)
			i.AddApiKey(&query)

			// Dynamically add query params
			switch etlConfig.Work {
			case Macros:
				query.Add("projection", "fundamental")
				query.Add("symbol", etlConfig.Symbol)
			case Medium:
				endDate := shared.NextDay(shared.Bod(time.Now()))
				startDate := endDate.AddDate(0, 0, -15)
				i.AddFetchPriceHistoryQuery(&query, PriceHistoryQuery{
					periodType:            "day",
					frequencyType:         "minute",
					frequency:             "30",
					startDate:             stringFormatDate(startDate),
					endDate:               stringFormatDate(endDate),
					needExtendedHoursData: "true",
				})
			case Short, Signals:
				endDate := shared.NextDay(shared.Bod(time.Now()))
				startDate := endDate.Add(time.Hour * -14)
				i.AddFetchPriceHistoryQuery(&query, PriceHistoryQuery{
					periodType:            "day",
					frequencyType:         "minute",
					frequency:             "15",
					startDate:             stringFormatDate(startDate),
					endDate:               stringFormatDate(endDate),
					needExtendedHoursData: "true",
				})
			}

			req.URL.RawQuery = query.Encode()
			resp, err := client.Do(req)
			if err != nil {
				panic(err)
			}

			defer resp.Body.Close()
			err = json.NewDecoder(resp.Body).Decode(&body)
			i.InsertResponse(etlConfig, resp, body)
			if resp.StatusCode >= 400 {
				if resp.StatusCode == 401 {
					return errors.New(UNAUTHORIZED)
				}
				if resp.StatusCode == 429 {
					return errors.New(SERVER_ERROR)
				}
				if resp.StatusCode >= 500 {
					return errors.New(SERVER_ERROR)
				}
				return errors.New("Api call failed with status code: " + strconv.Itoa(resp.StatusCode))
			}

			if err != nil {
				return err
			}

			return nil
		},
		retry.RetryIf(func(err error) bool {
			if err.Error() == SERVER_ERROR {
				return true
			}
			return false
		}),
		retry.Attempts(10),
		retry.OnRetry(func(n uint, err error) {
			log.Printf("Retrying request after error: %v", err)
		}),
		retry.DelayType(func(n uint, err error, config *retry.Config) time.Duration {
			fmt.Println("Server fails with: " + err.Error())
			// apply a default exponential back off strategy
			return retry.BackOffDelay(n, err, config) + (time.Millisecond * 750 * (time.Duration(n * 16 / 10))) + (1 * time.Second)
		}),
	)

	if err != nil {
		return ApiCallSuccess{}, err
	}

	log.Println("Api call success: ", etlConfig.Symbol)

	return CreateApiSuccess(body, etlConfig), nil
}

func (i *tdapiconfig) AddAuth(req *http.Request) {
	req.Header.Add("Authorization", "Bearer "+i.token.Fetch())
}

func (i *tdapiconfig) AddApiKey(query *url.Values) {
	query.Add("apikey", i.apikey)
}

type PriceHistoryQuery struct {
	periodType            string // default day
	frequencyType         string // ex minute, daily
	frequency             string // int ex 5
	startDate             string // unix mseconds int
	endDate               string // unix mseconds int
	needExtendedHoursData string // bool
}

func (i *tdapiconfig) AddFetchPriceHistoryQuery(query *url.Values, p PriceHistoryQuery) {
	query.Add("periodType", p.periodType)
	query.Add("frequencyType", p.frequencyType)
	query.Add("frequency", p.frequency)
	query.Add("startDate", p.startDate)
	query.Add("endDate", p.endDate)
	query.Add("needExtendedHoursData", p.needExtendedHoursData)

}

// log the api calls in table for transparency and analysis
func (i *tdapiconfig) InsertResponse(etlConfig EtlConfig, resp *http.Response, decodedBody interface{}) error {
	document := HttpResponsesDocument{
		EtlConfig: etlConfig,
		Response: APIResponse{
			Body:   decodedBody,
			Status: resp.StatusCode,
			Path:   resp.Request.URL.Path,
		},
	}
	err := i.mongo.ApiCalls.Cache(etlConfig, document)
	if err != nil {
		return err
	}
	return nil
}

type ApiCallSuccess struct {
	Body      map[string]interface{}
	etlConfig EtlConfig
}

func CreateApiSuccess(body map[string]interface{}, etlConfig EtlConfig) ApiCallSuccess {
	return ApiCallSuccess{Body: body, etlConfig: etlConfig}
}

func stringFormatDate(t time.Time) string {
	return strconv.FormatInt(t.Unix()*1000, 10)
}
