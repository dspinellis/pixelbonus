package i18n

import "strings"

type Catalog map[string]map[string]string

func NewCatalog() Catalog {
	return Catalog{
		"en": {
			"app.name":              "PixelBonus",
			"home.title":            "Your participation counts!",
			"home.student.title":    "I'm a participant",
			"home.student.body":     "Here you can redeem your personal QR code with only 2 clicks. Enter the code and your personal identifier and your participation will be instantly counted.",
			"home.instructor.title": "I'm an instructor",
			"home.instructor.body":  "With PixelBonus, you can reward active participation of the students by providing personal QR codes that they can redeem quickly and easily. Create an activity/course, print the codes and form rankings using several grading models.",
			"privacy.title":         "Privacy Policy",
		},
		"el": {
			"app.name":              "PixelBonus",
			"home.title":            "Η συμμετοχή σου μετράει!",
			"home.student.title":    "Είμαι συμμετέχων",
			"home.student.body":     "Εδώ μπορείς να εξαργυρώσεις το προσωπικό σου QR code με μόνο 2 κλικ. Συμπλήρωσε τον κωδικό και το προσωπικό σου αναγνωριστικό και η συμμετοχή σου θα καταμετρηθεί άμεσα.",
			"home.instructor.title": "Είμαι διδάσκων",
			"home.instructor.body":  "Με το PixelBonus μπορείτε να επιβραβεύετε την ενεργή συμμετοχή των φοιτητών προσφέροντας προσωπικούς κωδικούς QR που εξαργυρώνονται γρήγορα και εύκολα. Δημιουργήστε μάθημα, εκτυπώστε κωδικούς και σχηματίστε κατατάξεις με διάφορα μοντέλα βαθμολόγησης.",
			"privacy.title":         "Πολιτική Απορρήτου",
		},
	}
}

func ResolveLanguage(cookieValue string, acceptLanguage string) string {
	if cookieValue == "en" || cookieValue == "el" {
		return cookieValue
	}
	for _, part := range strings.Split(acceptLanguage, ",") {
		part = strings.ToLower(strings.TrimSpace(part))
		if strings.HasPrefix(part, "el") {
			return "el"
		}
		if strings.HasPrefix(part, "en") {
			return "en"
		}
	}
	return "en"
}

func (c Catalog) T(lang string, key string) string {
	if langMap, ok := c[lang]; ok {
		if text, ok := langMap[key]; ok {
			return text
		}
	}
	if text, ok := c["en"][key]; ok {
		return text
	}
	return key
}
