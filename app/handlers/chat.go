package handlers

import (
	"context"
	"regexp"
	"strings"
	"umrah/app/services"

	"github.com/gofiber/fiber/v2"
)

// Patterns that strongly indicate prompt injection or security probing attempts
var blockedPatterns = regexp.MustCompile(
	`(?i)(ignore (previous|all|above)|system prompt|jailbreak|act as|you are now|` +
		`pretend (to be|you)|override|bypass|reveal (your|the)|show (me )?(your|the) (code|env|config|key|secret|password|token)|` +
		`ls -|cat |rm -|chmod|curl |wget |exec|eval\(|<script|javascript:|DROP TABLE|SELECT \*|INSERT INTO|` +
		`\.env|api.?key|secret.?key|access.?token)`,
)

func HandleChat(c *fiber.Ctx) error {
	userMessage := strings.TrimSpace(c.FormValue("message"))

	// Empty message guard
	if userMessage == "" {
		return c.SendStatus(400)
	}

	// Max length guard (prevent token flooding)
	if len([]rune(userMessage)) > 500 {
		return c.Render("chat_bubble", fiber.Map{
			"UserMessage": userMessage[:100] + "...",
			"AIMessage":   "Maaf, pesan terlalu panjang. Mohon ajukan pertanyaan yang lebih singkat (maksimal 500 karakter).",
		})
	}

	// Block obvious injection/security probing
	if blockedPatterns.MatchString(userMessage) {
		return c.Render("chat_bubble", fiber.Map{
			"UserMessage": userMessage,
			"AIMessage":   "Maaf, saya hanya bisa membantu pencarian paket umrah. 🕌",
		})
	}

	// Call AI
	ctx := context.Background()
	aiMessage, err := services.ProcessChat(ctx, userMessage)
	if err != nil {
		aiMessage = "Mohon maaf, terjadi kesalahan. Silakan coba lagi."
	}

	return c.Render("chat_bubble", fiber.Map{
		"UserMessage": userMessage,
		"AIMessage":   aiMessage,
	})
}
