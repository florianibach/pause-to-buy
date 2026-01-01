package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"pausetobuye/internal/models"
	"pausetobuye/internal/repository"
	"pausetobuye/internal/services"
)

type Handler struct {
	repo      repository.Repository
	templates *template.Template
}

func NewHandler(repo repository.Repository) *Handler {
	funcMap := template.FuncMap{
		"formatPrice": func(price float64) string {
			return fmt.Sprintf("%.2f €", price)
		},
		"formatDate": func(t time.Time) string {
			return t.Format("02.01.2006")
		},
	}
	
	templates := template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))
	
	return &Handler{
		repo:      repo,
		templates: templates,
	}
}

func (h *Handler) HomeHandler(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.GetItems("")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	config, _ := h.repo.GetConfig()
	
	for i := range items {
		items[i].HoursToWork = items[i].Price / config.HourlyWage
		
		if items[i].Status == "waiting" {
			remaining := time.Until(items[i].WaitUntil)
			items[i].DaysRemaining = int(remaining.Hours() / 24)
			
			if remaining <= 0 {
				items[i].Status = "available"
				h.repo.UpdateItemStatus(items[i].ID, "available")
			}
		}
	}
	
	data := map[string]interface{}{
		"Title":  "Dashboard",
		"Items":  items,
		"Config": config,
	}
	
	h.templates.ExecuteTemplate(w, "index.html", data)
}

func (h *Handler) AddItemHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		h.templates.ExecuteTemplate(w, "add_form.html", nil)
		return
	}
	
	if r.Method == "POST" {
		r.ParseForm()
		
		price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
		waitDays, _ := strconv.Atoi(r.FormValue("wait_days"))
		
		item := &models.Item{
			Title:    r.FormValue("title"),
			Price:    price,
			Link:     r.FormValue("link"),
			Notes:    r.FormValue("notes"),
			Category: r.FormValue("category"),
			WaitDays: waitDays,
		}
		
		if err := h.repo.CreateItem(item); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		// Return updated item list for HTMX
		items, _ := h.repo.GetItems("")
		config, _ := h.repo.GetConfig()
		
		for i := range items {
			items[i].HoursToWork = items[i].Price / config.HourlyWage
			if items[i].Status == "waiting" {
				remaining := time.Until(items[i].WaitUntil)
				items[i].DaysRemaining = int(remaining.Hours() / 24)
			}
		}
		
		data := map[string]interface{}{
			"Items":  items,
			"Config": config,
		}
		
		h.templates.ExecuteTemplate(w, "items_list.html", data)
		return
	}
}

func (h *Handler) ItemHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/item/"):]
	id, _ := strconv.Atoi(idStr)
	
	item, err := h.repo.GetItemByID(id)
	if err != nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	
	config, _ := h.repo.GetConfig()
	item.HoursToWork = item.Price / config.HourlyWage
	
	data := map[string]interface{}{
		"Title": item.Title,
		"Item":  item,
	}
	
	h.templates.ExecuteTemplate(w, "item_detail.html", data)
}

func (h *Handler) UpdateStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	idStr := r.URL.Path[len("/update-status/"):]
	id, _ := strconv.Atoi(idStr)
	
	r.ParseForm()
	status := r.FormValue("status")
	
	h.repo.UpdateItemStatus(id, status)
	
	// Return updated item list
	items, _ := h.repo.GetItems("")
	config, _ := h.repo.GetConfig()
	
	for i := range items {
		items[i].HoursToWork = items[i].Price / config.HourlyWage
		if items[i].Status == "waiting" {
			remaining := time.Until(items[i].WaitUntil)
			items[i].DaysRemaining = int(remaining.Hours() / 24)
		}
	}
	
	data := map[string]interface{}{
		"Items":  items,
		"Config": config,
	}
	
	h.templates.ExecuteTemplate(w, "items_list.html", data)
}

func (h *Handler) StatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := h.repo.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	data := map[string]interface{}{
		"Title": "Statistics",
		"Stats": stats,
	}
	
	h.templates.ExecuteTemplate(w, "stats.html", data)
}

func (h *Handler) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		config, _ := h.repo.GetConfig()
		data := map[string]interface{}{
			"Title":  "Settings",
			"Config": config,
		}
		h.templates.ExecuteTemplate(w, "config.html", data)
		return
	}
	
	if r.Method == "POST" {
		r.ParseForm()
		wage, _ := strconv.ParseFloat(r.FormValue("hourly_wage"), 64)
		
		config := &models.Config{
			HourlyWage: wage,
			NtfyTopic:  r.FormValue("ntfy_topic"),
		}
		
		h.repo.UpdateConfig(config)
		
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)
		return
	}
}

func (h *Handler) CheckNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	config, _ := h.repo.GetConfig()
	
	if config.NtfyTopic == "" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	items, _ := h.repo.GetItems("available")
	
	for _, item := range items {
		services.SendNtfyNotification(config.NtfyTopic, item)
	}
	
	w.WriteHeader(http.StatusOK)
}