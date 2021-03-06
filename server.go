package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/DisgoOrg/disgohook"
	"github.com/DisgoOrg/disgohook/api"
	"github.com/line/line-bot-sdk-go/v7/linebot"
)

func main() {
	app, err := NewKitchenSink(
		os.Getenv("CHANNEL_SECRET"),
		os.Getenv("CHANNEL_TOKEN"),
		os.Getenv("APP_BASE_URL"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// serve /static/** files
	staticFileServer := http.FileServer(http.Dir("static"))
	http.HandleFunc("/static/", http.StripPrefix("/static/", staticFileServer).ServeHTTP)
	// serve /downloaded/** files
	downloadedFileServer := http.FileServer(http.Dir(app.downloadDir))
	http.HandleFunc("/downloaded/", http.StripPrefix("/downloaded/", downloadedFileServer).ServeHTTP)
	http.HandleFunc("/callback", app.Callback)
	// This is just a sample code.
	// For actually use, you must support HTTPS by using `ListenAndServeTLS`, reverse proxy or etc.
	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Fatal(err)
	}
}

// KitchenSink app
type KitchenSink struct {
	bot         *linebot.Client
	appBaseURL  string
	downloadDir string
}

// NewKitchenSink function
func NewKitchenSink(channelSecret, channelToken, appBaseURL string) (*KitchenSink, error) {
	apiEndpointBase := os.Getenv("ENDPOINT_BASE")
	if apiEndpointBase == "" {
		apiEndpointBase = linebot.APIEndpointBase
	}
	bot, err := linebot.New(
		channelSecret,
		channelToken,
		linebot.WithEndpointBase(apiEndpointBase), // Usually you omit this.
	)
	if err != nil {
		return nil, err
	}
	downloadDir := filepath.Join(filepath.Dir(os.Args[0]), "line-bot")
	_, err = os.Stat(downloadDir)
	if err != nil {
		if err := os.Mkdir(downloadDir, 0777); err != nil {
			return nil, err
		}
	}
	return &KitchenSink{
		bot:         bot,
		appBaseURL:  appBaseURL,
		downloadDir: downloadDir,
	}, nil
}

// Callback function for http server
func (app *KitchenSink) Callback(w http.ResponseWriter, r *http.Request) {
	events, err := app.bot.ParseRequest(r)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}
	for _, event := range events {
		log.Printf("Got event %v", event)
		if a := event.Source.GroupID; a == os.Getenv("GROUP_ID") || os.Getenv("GROUP_ID") == "" {
			switch event.Type {
			case linebot.EventTypeMessage:
				switch message := event.Message.(type) {
				case *linebot.TextMessage:
					if err := app.handleText(message, event.ReplyToken, event.Source); err != nil {
						log.Print(err)
					}
				case *linebot.ImageMessage:
					if err := app.handleImage(message, event.ReplyToken); err != nil {
						log.Print(err)
					}
				case *linebot.VideoMessage:
					if err := app.handleVideo(message, event.ReplyToken); err != nil {
						log.Print(err)
					}
				case *linebot.AudioMessage:
					if err := app.handleAudio(message, event.ReplyToken); err != nil {
						log.Print(err)
					}
				case *linebot.FileMessage:
					if err := app.handleFile(message, event.ReplyToken); err != nil {
						log.Print(err)
					}
				case *linebot.LocationMessage:
					if err := app.handleLocation(message, event.ReplyToken); err != nil {
						log.Print(err)
					}
				case *linebot.StickerMessage:
					if err := app.handleSticker(message, event.ReplyToken); err != nil {
						log.Print(err)
					}
				default:
					log.Printf("Unknown message: %v", message)
				}
			default:
				log.Printf("event: %v", event)
			}
		}
	}
}
func Chunks(s string, chunkSize int) []string {
	if len(s) == 0 {
		return nil
	}
	if chunkSize >= len(s) {
		return []string{s}
	}
	var chunks []string = make([]string, 0, (len(s)-1)/chunkSize+1)
	currentLen := 0
	currentStart := 0
	for i := range s {
		if currentLen == chunkSize {
			chunks = append(chunks, s[currentStart:i])
			currentLen = 0
			currentStart = i
		}
		currentLen++
	}
	chunks = append(chunks, s[currentStart:])
	return chunks
}
func (app *KitchenSink) handleText(message *linebot.TextMessage, replyToken string, source *linebot.EventSource) error {
	webhook, err := disgohook.NewWebhookClientByToken(nil, nil, os.Getenv("WEBHOOK_TOKEN"))
	if err != nil {
		fmt.Printf("failed to create webhook: %s", err)
		return err
	}
	webhook.SendContent("<@&744847472148349001>")
	a := Chunks(message.Text, 2000)
	for _, value := range a {
		webhook.SendContent(value)
	}
	return nil
}

func (app *KitchenSink) handleImage(message *linebot.ImageMessage, replyToken string) error {
	return app.handleHeavyContent(message.ID, func(originalContent *os.File) error {
		webhook, err := disgohook.NewWebhookClientByToken(nil, nil, os.Getenv("WEBHOOK_TOKEN"))
		if err != nil {
			fmt.Printf("failed to create webhook: %s", err)
			return err
		}
		reader, _ := os.Open(originalContent.Name())
		if _, err = webhook.SendMessage(api.NewWebhookMessageCreateBuilder().
			AddFile("image.jpeg", reader).
			Build(),
		); err != nil {
			fmt.Printf("failed to send webhook message: %s \n", err)
			return err
		}
		return nil
	})
}

func (app *KitchenSink) handleVideo(message *linebot.VideoMessage, replyToken string) error {
	return app.handleHeavyContent(message.ID, func(originalContent *os.File) error {
		webhook, err := disgohook.NewWebhookClientByToken(nil, nil, os.Getenv("WEBHOOK_TOKEN"))
		if err != nil {
			fmt.Printf("failed to create webhook: %s", err)
			return err
		}
		reader, _ := os.Open(originalContent.Name())
		if _, err = webhook.SendMessage(api.NewWebhookMessageCreateBuilder().
			AddFile("video.mp4", reader).
			Build(),
		); err != nil {
			fmt.Printf("failed to send webhook message: %s \n", err)
			return err
		}
		return nil
	})
}

func (app *KitchenSink) handleAudio(message *linebot.AudioMessage, replyToken string) error {
	return app.handleHeavyContent(message.ID, func(originalContent *os.File) error {
		webhook, err := disgohook.NewWebhookClientByToken(nil, nil, os.Getenv("WEBHOOK_TOKEN"))
		if err != nil {
			fmt.Printf("failed to create webhook: %s", err)
			return err
		}
		reader, _ := os.Open(originalContent.Name())
		if _, err = webhook.SendMessage(api.NewWebhookMessageCreateBuilder().
			AddFile("audio.m4a", reader).
			Build(),
		); err != nil {
			fmt.Printf("failed to send webhook message: %s \n", err)
			return err
		}
		return nil
	})
}

func (app *KitchenSink) handleFile(message *linebot.FileMessage, replyToken string) error {
	return app.handleHeavyContent(message.ID, func(originalContent *os.File) error {
		webhook, err := disgohook.NewWebhookClientByToken(nil, nil, os.Getenv("WEBHOOK_TOKEN"))
		if err != nil {
			fmt.Printf("failed to create webhook: %s", err)
			return err
		}
		reader, _ := os.Open(originalContent.Name())
		if _, err = webhook.SendMessage(api.NewWebhookMessageCreateBuilder().
			AddFile(message.FileName, reader).
			Build(),
		); err != nil {
			fmt.Printf("failed to send webhook message: %s \n", err)
			return err
		}
		return nil
	})
}

func (app *KitchenSink) handleLocation(message *linebot.LocationMessage, replyToken string) error {
	webhook, err := disgohook.NewWebhookClientByToken(nil, nil, os.Getenv("WEBHOOK_TOKEN"))
	if err != nil {
		fmt.Printf("failed to create webhook: %s", err)
		return err
	}
	if _, err := webhook.SendContent(message.Title + message.Address + fmt.Sprintf("%f", message.Latitude) + fmt.Sprintf("%f", message.Longitude)); err != nil {
		fmt.Printf("failed to send webhook message: %s \n", err)
		return err
	}
	return nil
}

func (app *KitchenSink) handleSticker(message *linebot.StickerMessage, replyToken string) error {
	webhook, err := disgohook.NewWebhookClientByToken(nil, nil, os.Getenv("WEBHOOK_TOKEN"))
	if err != nil {
		fmt.Printf("failed to create webhook: %s", err)
		return err
	}
	if _, err = webhook.SendContent(message.Keywords[0]); err != nil {
		fmt.Printf("failed to send webhook message: %s \n", err)
		return err
	}
	return nil
}

func (app *KitchenSink) handleHeavyContent(messageID string, callback func(*os.File) error) error {
	content, err := app.bot.GetMessageContent(messageID).Do()
	if err != nil {
		return err
	}
	defer content.Content.Close()
	log.Printf("Got file: %s", content.ContentType)
	originalContent, err := app.saveContent(content.Content)
	if err != nil {
		return err
	}
	return callback(originalContent)
}

func (app *KitchenSink) saveContent(content io.ReadCloser) (*os.File, error) {
	file, err := ioutil.TempFile(app.downloadDir, "")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	_, err = io.Copy(file, content)
	if err != nil {
		return nil, err
	}
	log.Printf("Saved %s", file.Name())
	return file, nil
}
