package templating

import (
	"fmt"
	"os"

	"github.com/HailoOSS/pongo2"

	"github.com/HailoOSS/go-hailo-lib/templating/filters"
)

// TemplateInfoSourceDetail contains details for use with rendering
type TemplateInfoSourceDetail interface {
	GetId() string
	GetRegulatoryArea() string
	GetLocale() string
}

// TemplateInfoSource is a source of data to be used in rendering using a template
type TemplateInfoSource interface {
	SetFormat(string)
	GetFormat(string) string
	GetTemplate() TemplateInfoSourceDetail
	GetTemplateData() map[string]interface{}
	PathOptions() ([]string, error)
}

// Templating wraps up a set of templates (TemplateSet) with a function to find the template files, and function to prepare the filters
type Templating struct {
	TemplateSet       *pongo2.TemplateSet
	assetInfo         AssetInfo
	FilterPreparation FilterPreparation
}

// AssetInfo defines the sugnatire of functions used to find the template files
type AssetInfo func(name string) (os.FileInfo, error)

// FilterPreparation defines the signature of functions used to prepare the filters
type FilterPreparation func(locale string, timezone string, currencyCode string) map[string]pongo2.FilterFunction

// TemplatePathFactory defines the signature of a function to get the paths to the templates
type TemplatePathFactory func(ctx TemplateContext) ([]string, error)

// TemplateContext an interface to serve as a generic type passed to the TemplatePathFactory - extend later?
type TemplateContext interface {
}

// NewTemplating creates a Templating instance - sets templatesDir and initializes template cache
func NewTemplating(fetcher pongo2.TemplateFetcher, af AssetInfo) *Templating {
	// Setup templates base dir, pongo2 call's it a set
	templateSet := pongo2.NewSet("cache templates")
	templateSet.TemplateFetcher = fetcher

	return &Templating{
		TemplateSet: templateSet,
		assetInfo:   af,
	}
}

// RenderTemplate renders a given templatePath with its data retruining the rendered template as a string
func (t *Templating) RenderTemplate(templatePath string, templateData pongo2.Context) (string, error) {
	tpl, err := t.TemplateSet.FromCache(templatePath)
	if err != nil {
		return "", fmt.Errorf("Error getting template '%s' from cache: %v", templateData, err)
	}

	return tpl.Execute(templateData)
}

// PrepareFilters a FilterPreparation function - prepares filters before rendering the template
func PrepareFilters(locale string, timezone string, currencyCode string) map[string]pongo2.FilterFunction {
	return map[string]pongo2.FilterFunction{
		"capitalize":                filters.Capitalize,
		"convertKilometersToMiles":  filters.ConvertKilometersToMiles,
		"currencySymbol":            filters.CurrencySymbol,
		"date":                      filters.SimpleDateFormatter(timezone),
		"escapeEntities":            filters.EscapeEntities,
		"formatCurrency":            filters.LocalizedFormatCurrency(currencyCode, locale),
		"formatCurrencyAmount":      filters.FormatCurrencyAmount(locale),
		"formatDecimal":             filters.FormatDecimalAmount(locale),
		"formatShortCurrencyAmount": filters.FormatShortCurrencyAmount(locale),
		"formatLocaleDate":          filters.LocalizedDateFormatter(locale, timezone),
		"raw":                       filters.Passthrough,
		"split":                     filters.Split,
		"unmarshalJson":             filters.UnmarshalJson,
		"maskAccountNumber":         filters.MaskAccountNumber,
		"lookup":                    filters.LookupMap,
		"insertSymbol":              filters.InsertSymbol,
	}
}

// FindTemplatePath Returns a template path based on the path-options that are given to it
func (t *Templating) FindTemplatePath(source TemplateInfoSource) (string, error) {
	// We get a prioritised list of paths - highest priority first
	paths, err := source.PathOptions()
	if err != nil {
		return "", err
	}

	// Now choose a path
	for _, relpath := range paths {
		info, _ := t.assetInfo(relpath)
		if info != nil {
			return relpath, nil
		}

	}

	//	return "", fmt.Errorf("Couldn't find template for [templateName=%s hob=%s locale=%s format=%s]", templateName, hob, locale, fileformat)
	return "", fmt.Errorf("Couldn't find template for [source: %+v]", source)
}

// FindAndRenderTemplate finds and renders a given TemplateSet and templatePath with the upplied data and context.
func (t *Templating) FindAndRenderTemplate(source TemplateInfoSource, templateData pongo2.Context) (string, error) {
	//	templatePath, err := t.FindTemplatePath(templateName, hob, locale, fileformat)
	templatePath, err := t.FindTemplatePath(source)
	if err != nil {
		return "", err
	}

	return t.RenderTemplate(templatePath, templateData)
}

// ExtractTemplateInfoFromRequest extracts template info from the supplied TemplateInfoSource, the info being: templateName, hob, locale and format
func (t *Templating) ExtractTemplateInfoFromRequest(request TemplateInfoSource) (templateName string, hob string, locale string, format string) {

	if tpl := request.GetTemplate(); tpl != nil {
		templateName = tpl.GetId()
		hob = tpl.GetRegulatoryArea()
		locale = tpl.GetLocale()
	}

	// Default to html
	format = request.GetFormat("html")

	return templateName, hob, locale, format
}

// RenderTemplateFromSource renders the supplied TemplateInfoSource using the 'self' templatData and context to a list of 'targets' output
// types (say, 'html', 'tsv' etc.).
func (t *Templating) RenderTemplateFromSource(request TemplateInfoSource, targets ...string) (map[string]string, error) {
	// Extract from the request the template name, hob, locale and fileformat
	//	templateName, hob, locale, _ := t.ExtractTemplateInfoFromRequest(request)
	// Get the data to substitute inside the template
	templateData := pongo2.Context(request.GetTemplateData())

	// Prepare filters for current locale, tz and money config
	currency := ExtractCurrency(templateData)
	timezone := ExtractTimezone(templateData)
	fp := PrepareFilters
	if t.FilterPreparation != nil {
		fp = t.FilterPreparation
	}
	templateData["_filters"] = fp(request.GetTemplate().GetLocale(), timezone, currency)

	// Render the template from the selected template with the extracted data
	content := map[string]string{}
	for _, target := range targets {

		// Alter the format to the target (makes this not parallelizable)
		request.SetFormat(target)

		out, err := t.FindAndRenderTemplate(request, templateData)
		if err != nil {
			return content, err
		}
		// @TODO(mark): This looks like it won't correctly pull the target?
		content[target] = out
	}

	return content, nil
}

// ExtractCurrency extracts the currency isocode from various locations in the supplied Context, and defaults to 'GBP'.
func ExtractCurrency(tplData pongo2.Context) string {
	for _, s := range []string{"job_currency", "hob_currency", "currency"} {
		if currency, ok := tplData[s]; ok && currency != nil {
			return currency.(string)
		}
	}

	// TODO: get from config
	// Default
	return "GBP"
}

// ExtractTimezone extracts the time-zone isocode from various location in the supplied Context, and defaults to 'Europe/London'.
func ExtractTimezone(tplData pongo2.Context) string {
	for _, s := range []string{"job_timezone", "hob_timezone", "timezone"} {
		if timezone, ok := tplData[s]; ok && timezone != nil {
			return timezone.(string)
		}
	}

	// TODO: get from config
	// Default
	return "Europe/London"
}

// Return a boolean if path exists
func fileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	return true
}
