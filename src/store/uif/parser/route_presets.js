const routePresetDefinitions = [
  { value: "openai", field: "geosite", values: ["openai"], label: "OpenAI", serviceLabel: "OpenAI（GeoSite）", service: true },
  { value: "google", field: "geosite", values: ["google"], label: "Google（包含 YouTube）", serviceLabel: "Google（GeoSite，包含 YouTube）", service: true },
  { value: "google-scholar", field: "geosite", values: ["google-scholar"], label: "Google Scholar", serviceLabel: "Google Scholar（GeoSite）", service: true },
  { value: "github", field: "geosite", values: ["github"], label: "GitHub", serviceLabel: "GitHub（GeoSite）", service: true },
  { value: "microsoft", field: "geosite", values: ["microsoft"], label: "Microsoft", serviceLabel: "Microsoft（GeoSite）", service: true },
  { value: "netflix", field: "geosite", values: ["netflix"], label: "Netflix", serviceLabel: "Netflix（GeoSite）", service: true },
  { value: "telegram", field: "geosite", values: ["telegram"], label: "Telegram", serviceLabel: "Telegram（GeoSite）", service: true },
  { value: "category-ads", field: "geosite", values: ["category-ads"], label: "广告", serviceLabel: "广告（GeoSite）", service: true },
  { value: "category-porn", field: "geosite", values: ["category-porn"], label: "成人内容", serviceLabel: "成人内容（GeoSite）", service: true },
  { value: "youtube", field: "domain", values: ["youtube.com"], label: "YouTube", serviceLabel: "YouTube（完全匹配）", service: true },
  { value: "grok", field: "domain_suffix", values: ["grok.com", "x.ai"], label: "Grok", serviceLabel: "Grok（后缀匹配）", service: true },
  { value: "github-domain", field: "domain", values: ["github.com"], label: "GitHub（完全匹配）" },
  { value: "domain-cn", field: "domain_suffix", values: [".cn"], label: ".cn" },
  { value: "domain-com", field: "domain_suffix", values: [".com"], label: ".com" },
]

const presetsForField = (field) => routePresetDefinitions
  .filter((item) => item.field === field)
  .reduce((result, item) => result.concat(item.values.map((value) => ({ value, label: item.label }))), [])

export const geoSitePresets = presetsForField("geosite")
export const domainPresets = presetsForField("domain")
export const domainSuffixPresets = presetsForField("domain_suffix")

export const servicePresets = routePresetDefinitions
  .filter((item) => item.service)
  .map((item) => ({
    value: item.value,
    label: item.serviceLabel,
    kind: item.field,
    values: item.values,
  }))

export { routePresetDefinitions }
