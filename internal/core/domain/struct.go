package domain

import (
	"time"
)

// SubdivisionGetInfo

type SubdivisionGetInfoReq struct {
	Customer struct {
		MobilePhone string `json:"mobilePhone"`
	} `json:"customer"`
	PointOfContact string `json:"pointOfContact"`
	ProductList    struct {
		CalculationDateTimeUtc string                      `json:"calculationDateTimeUtc"`
		Items                  []SubdivisionGetInfoReqItem `json:"items"`
	} `json:"productList"`
}

func (m *SubdivisionGetInfoReq) Encode(src *ImportModelReq) {
	m.PointOfContact = src.SubdivisionId
	m.ProductList.CalculationDateTimeUtc = (src.CalculationTime.UTC()).Format("2006-01-02 15:04:05")
	m.ProductList.Items = make([]SubdivisionGetInfoReqItem, 0, len(src.Products))

	for _, i := range src.Products {
		product := SubdivisionGetInfoReqItem{}
		product.Product.Ids.Mechtakz = i.ProductId
		product.BasePricePerItem = i.Price
		m.ProductList.Items = append(m.ProductList.Items, product)
	}
}

type SubdivisionGetInfoRep struct {
	Status      string `json:"status"`
	ProductList struct {
		ProcessingStatus string                       `json:"processingStatus"`
		Items            []*SubdivisionGetInfoRepItem `json:"items"`
	} `json:"productList"`
}

type SubdivisionGetInfoReqItem struct {
	Product struct {
		Ids struct {
			Mechtakz string `json:"mechtakz"`
		} `json:"ids"`
	} `json:"product"`
	BasePricePerItem float64 `json:"basePricePerItem"`
}

type SubdivisionGetInfoRepItem struct {
	Product struct {
		Ids struct {
			Mechtakz string `json:"mechtakz"`
		} `json:"ids"`
	} `json:"product"`
	BasePricePerItem  float64                       `json:"basePricePerItem"`
	PriceForCustomer  float64                       `json:"priceForCustomer"`
	AppliedPromotions []*SubdivisionGetInfoRepPromo `json:"appliedPromotions"`
	Placeholders      []*PlaceholderRep             `json:"placeholders"`
}

func (m *SubdivisionGetInfoRepItem) Decode() *ImportModelRep {
	resultPr := &ImportModelRep{
		FinalPrice: &FinalPrice{
			ProductId: m.Product.Ids.Mechtakz,
			Price:     m.PriceForCustomer,
		},
		Promotions:        []*Promo{},
		PromoPlaceholders: []*PromoPlaceholder{},
	}

	for _, p := range m.AppliedPromotions {
		resultPr.Promotions = append(resultPr.Promotions, &Promo{
			Id:         p.Promotion.Ids.MindboxId,
			ExternalId: p.Promotion.Ids.ExternalId,
			Type:       p.Type,
			Name:       p.Promotion.Name,
		})
	}

	for _, pl := range m.Placeholders {
		for _, content := range pl.Content {
			resultPr.PromoPlaceholders = append(resultPr.PromoPlaceholders, &PromoPlaceholder{
				PhId:    pl.IDs.ExternalID,
				PromoId: content.Promotion.IDs.MindboxID,
				Type:    content.Type,
				Message: content.Message,
			})
		}
	}

	return resultPr
}

type SubdivisionGetInfoRepPromo struct {
	Type      string `json:"type"`
	Promotion struct {
		Ids struct {
			MindboxId  int64  `json:"mindboxId"`
			ExternalId string `json:"externalId"`
		} `json:"ids"`
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"promotion"`
	GroupingKey string `json:"groupingKey"`
	BalanceType struct {
		Ids struct {
			SystemName string `json:"systemName"`
		} `json:"ids"`
		Name string `json:"name"`
	} `json:"balanceType"`
	Amount float64 `json:"amount"`
}

type PlaceholderRep struct {
	IDs struct {
		ExternalID string `json:"externalId"`
	} `json:"ids"`
	Content []struct {
		Type      string `json:"type"`
		Promotion struct {
			IDs struct {
				MindboxID int64 `json:"mindboxId"`
			} `json:"ids"`
			Name string `json:"name"`
			Type string `json:"type"`
		}
		Message string `json:"message"`
	} `json:"content"`
}

// GetPromotionsInfo

type ExportRepSt struct {
	Status       string `json:"status"`
	ExportID     string `json:"exportId"`
	ExportResult struct {
		ProcessingStatus string   `json:"processingStatus"`
		Urls             []string `json:"urls"`
	} `json:"exportResult"`
}

type PromotionsGetInfoRepSt struct {
	Promotions []PromotionSt `json:"promotions"`
}

type PromotionSt struct {
	Ids struct {
		ExternalID string `json:"externalId"`
	} `json:"ids"`
	Name             string  `json:"name"`
	Description      string  `json:"description"`
	StartDateTimeUtc *string `json:"startDateTimeUtc"`
	EndDateTimeUtc   *string `json:"endDateTimeUtc"`
	State            string  `json:"state"`
	CustomFields     struct {
		ShemaV1C string `json:"shemaV1C"`
	} `json:"customFields"`
}

// BASE STRUCTS
type ImportModelReq struct {
	SubdivisionId   string
	CalculationTime time.Time
	Products        []*BasePrice
}

type ImportModelRep struct {
	FinalPrice        *FinalPrice
	Promotions        []*Promo
	PromoPlaceholders []*PromoPlaceholder
}

type ImportPromotionsRep struct {
	ExternalID string
	Name       string
	SchemaID   string
	StartDate  *time.Time
	EndDate    *time.Time
}

type BasePrice struct {
	ProductId string
	Price     float64
}

type FinalPrice struct {
	ProductId string
	Price     float64
}

type Promo struct {
	Id         int64
	ExternalId string
	Type       string
	Name       string
	SchemaId   string
	StartDate  *time.Time
	EndDate    *time.Time
}

type PromoPlaceholder struct {
	PhId       string
	PromoId    int64
	Type       string
	Message    string
	ProductIds []string

	Promo *Promo
}
