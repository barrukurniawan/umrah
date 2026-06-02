package repositories

import (
	"log"

	"umrah/app/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	var err error
	DB, err = gorm.Open(sqlite.Open("data/umrah.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	DB.AutoMigrate(&models.Travel{}, &models.Package{})

	if DB.Migrator().HasTable(&models.Package{}) {
		var count int64
		DB.Model(&models.Package{}).Count(&count)
		if count == 0 {
			seedData()
		}
	}
}

func seedData() {
	travels := []models.Travel{
		{Name: "Al Haramain Travel", Rating: 4.8},
		{Name: "Nurul Iman Wisata", Rating: 4.6},
		{Name: "Madinah Tour & Travel", Rating: 4.7},
		{Name: "Baitullah Sejahtera", Rating: 4.5},
		{Name: "Zamzam Berkah Wisata", Rating: 4.4},
		{Name: "Hijrah Mulia Travel", Rating: 4.9},
		{Name: "Safa Marwah Tour", Rating: 4.3},
		{Name: "Arafah Indah Travel", Rating: 4.6},
	}
	DB.Create(&travels)

	packages := []models.Package{
		{TravelID: 1, Name: "Paket Reguler", Price: 25000000, HotelDistance: 1000, IsFamily: false, IsSenior: false, SunnahScore: 7, GroupSize: 50, IsCheap: true, IsNearHaram: false, IsKajian: false, IsSunnah: false, IsKidFriendly: false, IsSeniorFriendly: false},
		{TravelID: 1, Name: "Paket Hemat", Price: 23000000, HotelDistance: 1200, IsFamily: false, IsSenior: false, SunnahScore: 6, GroupSize: 60, IsCheap: true, IsNearHaram: false, IsKajian: false, IsSunnah: false, IsKidFriendly: false, IsSeniorFriendly: false},
		{TravelID: 1, Name: "Paket Keluarga", Price: 28000000, HotelDistance: 600, IsFamily: true, IsSenior: false, SunnahScore: 8, GroupSize: 30, IsCheap: false, IsNearHaram: true, IsKajian: true, IsSunnah: false, IsKidFriendly: true, IsSeniorFriendly: false},
		{TravelID: 2, Name: "Paket Ekonomi", Price: 24000000, HotelDistance: 900, IsFamily: false, IsSenior: false, SunnahScore: 7, GroupSize: 45, IsCheap: true, IsNearHaram: false, IsKajian: false, IsSunnah: true, IsKidFriendly: false, IsSeniorFriendly: false},
		{TravelID: 2, Name: "Paket Plus", Price: 30000000, HotelDistance: 400, IsFamily: true, IsSenior: false, SunnahScore: 8, GroupSize: 25, IsCheap: false, IsNearHaram: true, IsKajian: true, IsSunnah: false, IsKidFriendly: true, IsSeniorFriendly: false},
		{TravelID: 2, Name: "Paket Premium", Price: 35000000, HotelDistance: 200, IsFamily: false, IsSenior: false, SunnahScore: 9, GroupSize: 15, IsCheap: false, IsNearHaram: true, IsKajian: true, IsSunnah: true, IsKidFriendly: false, IsSeniorFriendly: false},
		{TravelID: 3, Name: "Paket Standar", Price: 27000000, HotelDistance: 700, IsFamily: false, IsSenior: true, SunnahScore: 8, GroupSize: 35, IsCheap: false, IsNearHaram: false, IsKajian: false, IsSunnah: false, IsKidFriendly: false, IsSeniorFriendly: true},
		{TravelID: 3, Name: "Paket Lansia", Price: 32000000, HotelDistance: 300, IsFamily: false, IsSenior: true, SunnahScore: 9, GroupSize: 20, IsCheap: false, IsNearHaram: true, IsKajian: true, IsSunnah: true, IsKidFriendly: false, IsSeniorFriendly: true},
		{TravelID: 3, Name: "Paket VIP", Price: 40000000, HotelDistance: 100, IsFamily: true, IsSenior: true, SunnahScore: 10, GroupSize: 10, IsCheap: false, IsNearHaram: true, IsKajian: true, IsSunnah: true, IsKidFriendly: true, IsSeniorFriendly: true},
		{TravelID: 4, Name: "Paket Ekonomi Syariah", Price: 26000000, HotelDistance: 800, IsFamily: false, IsSenior: false, SunnahScore: 8, GroupSize: 40, IsCheap: true, IsNearHaram: false, IsKajian: true, IsSunnah: true, IsKidFriendly: false, IsSeniorFriendly: false},
		{TravelID: 4, Name: "Paket Eksklusif", Price: 38000000, HotelDistance: 150, IsFamily: false, IsSenior: false, SunnahScore: 10, GroupSize: 12, IsCheap: false, IsNearHaram: true, IsKajian: true, IsSunnah: true, IsKidFriendly: false, IsSeniorFriendly: false},
		{TravelID: 5, Name: "Paket Ramah Kantong", Price: 22000000, HotelDistance: 1100, IsFamily: false, IsSenior: false, SunnahScore: 5, GroupSize: 55, IsCheap: true, IsNearHaram: false, IsKajian: false, IsSunnah: false, IsKidFriendly: false, IsSeniorFriendly: false},
		{TravelID: 5, Name: "Paket Keluarga Bahagia", Price: 29000000, HotelDistance: 500, IsFamily: true, IsSenior: false, SunnahScore: 8, GroupSize: 28, IsCheap: false, IsNearHaram: true, IsKajian: true, IsSunnah: false, IsKidFriendly: true, IsSeniorFriendly: false},
		{TravelID: 6, Name: "Paket Golden", Price: 34000000, HotelDistance: 250, IsFamily: false, IsSenior: true, SunnahScore: 9, GroupSize: 18, IsCheap: false, IsNearHaram: true, IsKajian: true, IsSunnah: true, IsKidFriendly: false, IsSeniorFriendly: true},
		{TravelID: 6, Name: "Paket Platinum", Price: 42000000, HotelDistance: 100, IsFamily: true, IsSenior: true, SunnahScore: 10, GroupSize: 8, IsCheap: false, IsNearHaram: true, IsKajian: true, IsSunnah: true, IsKidFriendly: true, IsSeniorFriendly: true},
		{TravelID: 7, Name: "Paket Murah Berkah", Price: 21000000, HotelDistance: 1300, IsFamily: false, IsSenior: false, SunnahScore: 6, GroupSize: 65, IsCheap: true, IsNearHaram: false, IsKajian: false, IsSunnah: false, IsKidFriendly: false, IsSeniorFriendly: false},
		{TravelID: 7, Name: "Paket Pertengahan", Price: 28000000, HotelDistance: 550, IsFamily: true, IsSenior: false, SunnahScore: 7, GroupSize: 30, IsCheap: false, IsNearHaram: false, IsKajian: true, IsSunnah: false, IsKidFriendly: true, IsSeniorFriendly: false},
		{TravelID: 8, Name: "Paket Tabungan", Price: 25000000, HotelDistance: 750, IsFamily: false, IsSenior: false, SunnahScore: 7, GroupSize: 40, IsCheap: true, IsNearHaram: false, IsKajian: false, IsSunnah: true, IsKidFriendly: false, IsSeniorFriendly: false},
		{TravelID: 8, Name: "Paket Nyaman", Price: 31000000, HotelDistance: 350, IsFamily: true, IsSenior: false, SunnahScore: 8, GroupSize: 22, IsCheap: false, IsNearHaram: true, IsKajian: true, IsSunnah: false, IsKidFriendly: true, IsSeniorFriendly: false},
		{TravelID: 8, Name: "Paket Exclusive", Price: 39000000, HotelDistance: 120, IsFamily: true, IsSenior: true, SunnahScore: 10, GroupSize: 10, IsCheap: false, IsNearHaram: true, IsKajian: true, IsSunnah: true, IsKidFriendly: true, IsSeniorFriendly: true},
	}
	DB.Create(&packages)

	log.Println("Database seeded with", len(travels), "travels and", len(packages), "packages")
}
