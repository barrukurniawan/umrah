package models

import "gorm.io/gorm"

type Travel struct {
	gorm.Model
	Name   string  `json:"name"`
	Rating float64 `json:"rating"`
	Packages []Package `gorm:"foreignKey:TravelID"`
}

type Package struct {
	gorm.Model
	TravelID      uint   `json:"travel_id"`
	Name          string `json:"name"`
	Price         int    `json:"price"`
	HotelDistance int    `json:"hotel_distance"`
	IsFamily      bool   `json:"is_family"`
	IsSenior      bool   `json:"is_senior"`
	SunnahScore   int    `json:"sunnah_score"`
	GroupSize     int    `json:"group_size"`
	IsCheap       bool   `json:"is_cheap"`
	IsNearHaram   bool   `json:"is_near_haram"`
	IsKajian      bool   `json:"is_kajian"`
	IsSunnah      bool   `json:"is_sunnah"`
	IsKidFriendly bool   `json:"is_kid_friendly"`
	IsSeniorFriendly bool `json:"is_senior_friendly"`
	Travel        Travel `gorm:"foreignKey:TravelID"`
}
