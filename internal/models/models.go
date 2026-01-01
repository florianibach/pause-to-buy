package models

import "time"

type Item struct {
	ID             int
	Title          string
	Price          float64
	Link           string
	Notes          string
	Category       string
	WaitDays       int
	CreatedAt      time.Time
	WaitUntil      time.Time
	Status         string // waiting, available, bought, skipped
	HoursToWork    float64
	DaysRemaining  int
}

type Stats struct {
	TotalItems   int
	ItemsSkipped int
	ItemsBought  int
	MoneySaved   float64
	MoneySpent   float64
	TopCategory  string
}

type Config struct {
	HourlyWage float64
	NtfyTopic  string
}