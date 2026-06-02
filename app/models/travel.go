package models

import (
	"time"

	"gorm.io/gorm"
)

type Travel struct {
	gorm.Model
	Name     string    `json:"name"`
	Rating   float64   `json:"rating"`
	Packages []Package `gorm:"foreignKey:TravelID"`
}

type Package struct {
	gorm.Model
	TravelID         uint            `json:"travel_id"`
	Name             string          `json:"name"`
	Price            int             `json:"price"`
	HotelDistance    int             `json:"hotel_distance"`
	Duration         int             `json:"duration"`
	Airline          string          `json:"airline"`
	IsDirect         bool            `json:"is_direct"`
	DownPayment      int             `json:"down_payment"`
	PaymentDeadline  string          `json:"payment_deadline"`
	Guide            string          `json:"guide"`
	Facilities       string          `json:"facilities"`
	IsFamily         bool            `json:"is_family"`
	IsSenior         bool            `json:"is_senior"`
	SunnahScore      int             `json:"sunnah_score"`
	GroupSize        int             `json:"group_size"`
	IsCheap          bool            `json:"is_cheap"`
	IsNearHaram      bool            `json:"is_near_haram"`
	IsKajian         bool            `json:"is_kajian"`
	IsSunnah         bool            `json:"is_sunnah"`
	IsKidFriendly    bool            `json:"is_kid_friendly"`
	IsSeniorFriendly bool            `json:"is_senior_friendly"`
	Travel           Travel          `gorm:"foreignKey:TravelID"`
	Details          []DetailPackage `gorm:"foreignKey:PackageID"`
}

type DetailPackage struct {
	gorm.Model
	PackageID         uint    `json:"package_id"`
	DepartureDate     string  `json:"departure_date"`
	ReturnDate        string  `json:"return_date"`
	HotelMakkah       string  `json:"hotel_makkah"`
	HotelMadinah      string  `json:"hotel_madinah"`
	StarsMakkah       int     `json:"stars_makkah"`
	StarsMadinah      int     `json:"stars_madinah"`
	RoomType          string  `json:"room_type"`
	TotalQuota        int     `json:"total_quota"`
	AvailableQuota    int     `json:"available_quota"`
	DepartureLocation string  `json:"departure_location"`
	AddonTriple       int     `json:"addon_triple"`
	AddonDouble       int     `json:"addon_double"`
	Guide             string  `json:"guide"`
	Package           Package `gorm:"foreignKey:PackageID"`
}

func (d DetailPackage) DPDeadline() string {
	t, err := time.Parse("2006-01-02", d.DepartureDate)
	if err != nil {
		return ""
	}
	return t.AddDate(0, -1, 0).Format("2006-01-02")
}
