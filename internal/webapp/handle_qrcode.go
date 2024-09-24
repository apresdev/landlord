package webapp

import (
	"encoding/base64"
	"net/http"

	"github.com/skip2/go-qrcode"

	"github.com/gin-gonic/gin"
)

func handleQRCode(a App, c *gin.Context) error {
	uri := c.Query("uri")
	if uri == "" {
		c.String(http.StatusBadRequest, "missing uri")
		return nil
	}
	b, err := base64.URLEncoding.DecodeString(uri)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid uri")
		return nil
	}
	png, err := qrcode.Encode(string(b), qrcode.Medium, 256)
	if err != nil {
		return err
	}
	c.Data(http.StatusOK, "image/png", png)
	return nil
}
