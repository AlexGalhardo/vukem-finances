package controllers

import (
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/pages"
)

// Home renders the Home page component (resources/js/pages/Home).
func Home(c *web.Ctx) error { return pages.Home.Render(c, struct{}{}) }
