package fastapi

import "github.com/gofiber/fiber/v2"

func DefaultCORS(c *fiber.Ctx) error {
	c.Set("Access-Control-Allow-Origin", "*")
	c.Set("Access-Control-Allow-Headers", "*")
	c.Set("Access-Control-Allow-Credentials", "false")
	c.Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS,DELETE,PATCH")

	if c.Method() == fiber.MethodOptions {
		c.Status(fiber.StatusOK)
		return nil
	}
	return c.Next()
}
