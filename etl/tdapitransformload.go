package etl

import (
	"context"
	"encoding/json"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TransformLoad(mongo MongoController, resp ApiCallSuccess) error {
	switch resp.etlConfig.Work {
	case Macros:
		var instrument Instrument
		b, err := json.Marshal(resp.Body[resp.etlConfig.Symbol])
		json.Unmarshal(b, &instrument)

		*instrument.Fundamental.MarketCap = Round(*instrument.Fundamental.MarketCap)

		// we exit earlier and save a smaller payload if marketcap is less than 500 million
		if *instrument.Fundamental.MarketCap < 500 {
			_, err := mongo.Macros.UpdateOne(context.TODO(),
				bson.M{"symbol": resp.etlConfig.Symbol},
				bson.M{"$set": bson.M{"marketCap": instrument.Fundamental.MarketCap}},
				options.Update().SetUpsert(true))
			if err != nil {
				return err
			}
			break // exit out of switch
		}

		{
			*instrument.Fundamental.High52 = Round(*instrument.Fundamental.High52)
			*instrument.Fundamental.Low52 = Round(*instrument.Fundamental.Low52)
			*instrument.Fundamental.DividendAmount = Round(*instrument.Fundamental.DividendAmount)
			*instrument.Fundamental.DividendYield = Round(*instrument.Fundamental.DividendYield)
			*instrument.Fundamental.PeRatio = Round(*instrument.Fundamental.PeRatio)
			*instrument.Fundamental.PegRatio = Round(*instrument.Fundamental.PegRatio)
			*instrument.Fundamental.PbRatio = Round(*instrument.Fundamental.PbRatio)
			*instrument.Fundamental.PrRatio = Round(*instrument.Fundamental.PrRatio)
			*instrument.Fundamental.PcfRatio = Round(*instrument.Fundamental.PcfRatio)
			*instrument.Fundamental.GrossMarginTTM = Round(*instrument.Fundamental.GrossMarginTTM)
			*instrument.Fundamental.GrossMarginMRQ = Round(*instrument.Fundamental.GrossMarginMRQ)
			*instrument.Fundamental.NetProfitMarginTTM = Round(*instrument.Fundamental.NetProfitMarginTTM)
			*instrument.Fundamental.NetProfitMarginMRQ = Round(*instrument.Fundamental.NetProfitMarginMRQ)
			*instrument.Fundamental.OperatingMarginTTM = Round(*instrument.Fundamental.OperatingMarginTTM)
			*instrument.Fundamental.OperatingMarginMRQ = Round(*instrument.Fundamental.OperatingMarginMRQ)
			*instrument.Fundamental.ReturnOnEquity = Round(*instrument.Fundamental.ReturnOnEquity)
			*instrument.Fundamental.ReturnOnAssets = Round(*instrument.Fundamental.ReturnOnAssets)
			*instrument.Fundamental.ReturnOnInvestment = Round(*instrument.Fundamental.ReturnOnInvestment)
			*instrument.Fundamental.QuickRatio = Round(*instrument.Fundamental.QuickRatio)
			*instrument.Fundamental.CurrentRatio = Round(*instrument.Fundamental.CurrentRatio)
			*instrument.Fundamental.InterestCoverage = Round(*instrument.Fundamental.InterestCoverage)
			*instrument.Fundamental.TotalDebtToCapital = Round(*instrument.Fundamental.TotalDebtToCapital)
			*instrument.Fundamental.LtDebtToEquity = Round(*instrument.Fundamental.LtDebtToEquity)
			*instrument.Fundamental.TotalDebtToEquity = Round(*instrument.Fundamental.TotalDebtToEquity)
			*instrument.Fundamental.EpsTTM = Round(*instrument.Fundamental.EpsTTM)
			*instrument.Fundamental.EpsChangePercentTTM = Round(*instrument.Fundamental.EpsChangePercentTTM)
			*instrument.Fundamental.EpsChangeYear = Round(*instrument.Fundamental.EpsChangeYear)
			*instrument.Fundamental.RevChangeTTM = Round(*instrument.Fundamental.RevChangeTTM)
			*instrument.Fundamental.MarketCapFloat = Round(*instrument.Fundamental.MarketCapFloat)
			*instrument.Fundamental.BookValuePerShare = Round(*instrument.Fundamental.BookValuePerShare)
			*instrument.Fundamental.DividendPayAmount = Round(*instrument.Fundamental.DividendPayAmount)
			*instrument.Fundamental.Beta = Round(*instrument.Fundamental.Beta)
		}

		_, err = mongo.Macros.UpdateOne(context.TODO(),
			bson.M{"symbol": resp.etlConfig.Symbol},
			bson.M{"$set": instrument},
			options.Update().SetUpsert(true))
		if err != nil {
			return err
		}
	case Medium:
		candles, err := respBodyToPriceHistory(resp.Body)
		_, err = mongo.Medium.UpdateOne(context.TODO(),
			bson.M{"symbol": candles.Symbol},
			bson.M{"$set": candles},
			options.Update().SetUpsert(true))
		if err != nil {
			return err
		}
	case Short:
		candles, err := respBodyToPriceHistory(resp.Body)
		_, err = mongo.Short.UpdateOne(context.TODO(),
			bson.M{"symbol": candles.Symbol},
			bson.M{"$set": candles},
			options.Update().SetUpsert(true))
		if err != nil {
			return err
		}
	case Signals:
		candles, err := respBodyToPriceHistory(resp.Body)
		_, err = mongo.Signals.UpdateOne(context.TODO(),
			bson.M{"symbol": candles.Symbol},
			bson.M{"$set": candles},
			options.Update().SetUpsert(true))
		if err != nil {
			return err
		}
	}

	err := mongo.ApiQueue.Remove(resp.etlConfig)
	if err != nil {
		return err
	}

	log.Println("Transform load success: ", resp.etlConfig.Symbol)

	return nil
}

func respBodyToPriceHistory(body interface{}) (*PriceHistory, error) {
	var ph PriceHistory
	b, err := json.Marshal(body)
	json.Unmarshal(b, &ph)

	candles, err := calculatePriceHistory(ph)
	if err != nil {
		return nil, err
	}
	return candles, nil
}

type Instrument struct {
	ID          *primitive.ObjectID `json:"id,omitempty"  bson:"_id,omitempty"`
	Fundamental Fundamental         `json:"fundamental" bson:"fundamental"`
	Cusip       *string             `json:"cusip" bson:"cusip"`
	Symbol      *string             `json:"symbol" bson:"symbol"`
	Description *string             `json:"description" bson:"description"`
	Exchange    *string             `json:"exchange" bson:"exchange"`
	AssetType   *string             `json:"assetType,omitempty" bson:"assetType,omitempty"`
}

type Fundamental struct {
	Symbol              *string  `json:"symbol" bson:"symbol"`
	High52              *float64 `json:"high52" bson:"high52"`
	Low52               *float64 `json:"low52" bson:"low52"`
	DividendAmount      *float64 `json:"dividendAmount" bson:"dividendAmount"`
	DividendYield       *float64 `json:"dividendYield" bson:"dividendYield"`
	DividendDate        *string  `json:"dividendDate" bson:"dividendDate"`
	PeRatio             *float64 `json:"peRatio" bson:"peRatio"`
	PegRatio            *float64 `json:"pegRatio" bson:"pegRatio"`
	PbRatio             *float64 `json:"pbRatio" bson:"pbRatio"`
	PrRatio             *float64 `json:"prRatio" bson:"prRatio"`
	PcfRatio            *float64 `json:"pcfRatio" bson:"pcfRatio"`
	GrossMarginTTM      *float64 `json:"grossMarginTTM" bson:"grossMarginTTM"`
	GrossMarginMRQ      *float64 `json:"grossMarginMRQ" bson:"grossMarginMRQ"`
	NetProfitMarginTTM  *float64 `json:"netProfitMarginTTM" bson:"netProfitMarginTTM"`
	NetProfitMarginMRQ  *float64 `json:"netProfitMarginMRQ" bson:"netProfitMarginMRQ"`
	OperatingMarginTTM  *float64 `json:"operatingMarginTTM" bson:"operatingMarginTTM"`
	OperatingMarginMRQ  *float64 `json:"operatingMarginMRQ" bson:"operatingMarginMRQ"`
	ReturnOnEquity      *float64 `json:"returnOnEquity" bson:"returnOnEquity"`
	ReturnOnAssets      *float64 `json:"returnOnAssets" bson:"returnOnAssets"`
	ReturnOnInvestment  *float64 `json:"returnOnInvestment" bson:"returnOnInvestment"`
	QuickRatio          *float64 `json:"quickRatio" bson:"quickRatio"`
	CurrentRatio        *float64 `json:"currentRatio" bson:"currentRatio"`
	InterestCoverage    *float64 `json:"interestCoverage" bson:"interestCoverage"`
	TotalDebtToCapital  *float64 `json:"totalDebtToCapital" bson:"totalDebtToCapital"`
	LtDebtToEquity      *float64 `json:"ltDebtToEquity" bson:"ltDebtToEquity"`
	TotalDebtToEquity   *float64 `json:"totalDebtToEquity" bson:"totalDebtToEquity"`
	EpsTTM              *float64 `json:"epsTTM" bson:"epsTTM"`
	EpsChangePercentTTM *float64 `json:"epsChangePercentTTM" bson:"epsChangePercentTTM"`
	EpsChangeYear       *float64 `json:"epsChangeYear" bson:"epsChangeYear"`
	EpsChange           *int     `json:"epsChange" bson:"epsChange"`
	RevChangeYear       *int     `json:"revChangeYear" bson:"revChangeYear"`
	RevChangeTTM        *float64 `json:"revChangeTTM" bson:"revChangeTTM"`
	RevChangeIn         *int     `json:"revChangeIn" bson:"revChangeIn"`
	SharesOutstanding   *float64 `json:"sharesOutstanding" bson:"sharesOutstanding"`
	MarketCapFloat      *float64 `json:"marketCapFloat" bson:"marketCapFloat"`
	MarketCap           *float64 `json:"marketCap" bson:"marketCap"`
	BookValuePerShare   *float64 `json:"bookValuePerShare" bson:"bookValuePerShare"`
	ShortIntToFloat     *int     `json:"shortIntToFloat" bson:"shortIntToFloat"`
	ShortIntDayToCover  *int     `json:"shortIntDayToCover" bson:"shortIntDayToCover"`
	DivGrowthRate3Year  *int     `json:"divGrowthRate3Year" bson:"divGrowthRate3Year"`
	DividendPayAmount   *float64 `json:"dividendPayAmount" bson:"dividendPayAmount"`
	DividendPayDate     *string  `json:"dividendPayDate" bson:"dividendPayDate"`
	Beta                *float64 `json:"beta" bson:"beta"`
	Vol1DayAvg          *float64 `json:"vol1DayAvg" bson:"vol1DayAvg"`
	Vol10DayAvg         *float64 `json:"vol10DayAvg" bson:"vol10DayAvg"`
	Vol3MonthAvg        *float64 `json:"vol3MonthAvg" bson:"vol3MonthAvg"`
}
