package services

import (
	"fmt"
	"net/http"
	"pausetobuye/internal/models"
	"time"
)

func SendNtfyNotification(topic string, item models.Item) error {
	message := fmt.Sprintf("⏰ Wait time over: %s (%.2f €)", item.Title, item.Price)
	
	req, err := http.NewRequest("POST", "https://ntfy.sh/"+topic, nil)
	if err != nil {
		return err
	}
	
	req.Header.Set("Title", "PauseToBuye")
	req.Header.Set("Message", message)
	req.Header.Set("Priority", "default")
	
	client := &http.Client{Timeout: 10 * time.Second}
	_, err = client.Do(req)
	
	return err
}