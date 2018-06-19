package main

import (
	"net/http"
	"time"

	"html/template"

	"github.com/go-mego/html"
	"github.com/go-mego/i18n"
	"github.com/go-mego/mego"
)

func main() {
	e := mego.Default()
	e.Use(html.New(&html.Options{
		Directory: "./templates",
		Functions: template.FuncMap{
			"i18n": func(a, b string) string {
				return a + b
			},
		},
		Templates: []*html.Template{
			{
				Name: "main",
				File: "main",
			},
		},
	}))
	e.GET("/", i18n.New(&i18n.Options{
		Directory:        "./locales",
		FallbackLanguage: "zh-TW",
	}), func(c *mego.Context, r *html.Renderer, l *i18n.Locale) {
		tomorrow := time.Now().AddDate(0, 0, 1)

		err := r.Render(http.StatusOK, "main", html.H{
			"Messages": html.H{
				"Welcome": l.Get("messages.welcome"),
				"Notice": l.Get("messages.notice", map[string]interface{}{
					"ip":       c.ClientIP(),
					"language": l.Language(),
				}),
				"Time":    l.Get("messages.time", time.Now(), time.Until(time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, tomorrow.Location()))),
				"Apples":  l.Get("messages.apples", 0),
				"Bananas": l.Get("messages.bananas", 30),
			},
		})

		/*err := r.Render(http.StatusOK, "main", html.H{
			"Notice": map[string]interface{}{
				"ip":       c.ClientIP(),
				"language": l.Language(),
			},
			"Time":      time.Now(),
			"Remaining": time.Until(time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, tomorrow.Location())),
		})*/
		if err != nil {
			panic(err)
		}
	})
	e.Run()
}
