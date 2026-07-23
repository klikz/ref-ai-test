package api

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/store"
)

func (s *ServerModel) BrandsGetAll(c *gin.Context) {
	data, err := s.Store.Repo().BrandList()
	if err != nil {
		s.Utils.SendError(c, err, "BrandsGetAll", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) BrandLogosGetAll(c *gin.Context) {
	data, err := s.Store.Repo().BrandLogosAll()
	if err != nil {
		s.Utils.SendError(c, err, "BrandLogosGetAll", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) BrandLogoUpsert(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "BrandLogoUpsert: ReadBody", "")
		return
	}

	brand, _ := jsonMap["brand"].(string)
	brand = strings.TrimSpace(brand)
	if brand == "" {
		s.Utils.SendError(c, errors.New("brand is required"), "BrandLogoUpsert", "")
		return
	}
	logoSrc, _ := jsonMap["logo_src"].(string)
	logoSrc = strings.TrimSpace(logoSrc)
	if logoSrc == "" {
		s.Utils.SendError(c, errors.New("logo_src is required"), "BrandLogoUpsert", "")
		return
	}

	userID := c.GetInt("user_id")
	if err := s.Store.Repo().BrandLogoUpsert(brand, logoSrc, userID); err != nil {
		s.Utils.SendError(c, err, "BrandLogoUpsert", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) BrandLogoDelete(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "BrandLogoDelete: ReadBody", "")
		return
	}

	brand, _ := jsonMap["brand"].(string)
	brand = strings.TrimSpace(brand)
	if brand == "" {
		s.Utils.SendError(c, errors.New("brand is required"), "BrandLogoDelete", "")
		return
	}
	userID := c.GetInt("user_id")
	if err := s.Store.Repo().BrandLogoDelete(brand, userID); err != nil {
		s.Utils.SendError(c, err, "BrandLogoDelete", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) applyBrandLogoData(modelBrand string, printData map[string]any) error {
	logos, err := s.Store.Repo().BrandLogosAll()
	if err != nil {
		return err
	}
	printData["brand_logo"] = store.ResolveBrandLogo(modelBrand, logos)
	return nil
}
