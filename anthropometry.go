/*Package anthropometry facilitates the making of anthropometric measurements and
conclusions. It is meant to work in conjunction with the results that come out the computer vision
models, as well as the subject's weight, age, gender, and other biographical information */
package anthropometry

import (
	"fmt"
	"github.com/frahman5/danso-backend/services/config"
	"math"
	"strconv"
)

// Anthropometry is a struct that holds configs needed for the various funcitons
// in this package, and also is the type that holds all the functions in this package
type Anthropometry struct {
	// Config is a struct that holds all the application's configurations
	Config config.Config
}

// Declare a global whoData struct to use in all functions
var globalWhoData whoData

const (
	// Anthropometric Indicator Names
	HCFA = "HeadCircumferenceForAge"
	HFA  = "HeightForAge"
	WFA  = "WeightForAge"
	WFH  = "WeightForHeight"

	// Categorizations for Head Circumference for Age
	HCFALessNeg3   = "Severe Microcephaly"
	HCFANeg3ToNeg2 = "Microcephaly"
	HCFANeg2To3    = "Normal"
	HCFAGreater3   = "Macrocephaly (not related to nutritional status)"

	// Categorizations for Height for Age
	HFALessNeg3   = "Severe Stunting"
	HFANeg3ToNeg2 = "Moderate Stunting"
	HFANeg2To3    = "Normal"
	HFAGreater3   = "Extreme Tallness (not a nutrition related concern)"

	// Categorizations for Weight For Age
	WFALessNeg3   = "Severe Underweight"
	WFANeg3ToNeg2 = "Moderate Underweight"
	WFANeg2To1    = "Normal"
	WFAGreater1   = "Out of Range. See Weight For Length/Height"

	// Categorizations for Weight for Height
	WFHLessNeg3   = "Severe Acute Malnutrition (SAM)"
	WFHNeg3ToNeg2 = "Moderate Acute Malnutrition (MAM)"
	WFHNeg2To1    = "Normal"
	WFH1To2       = "At risk for overweight"
	WFH2To3       = "Overweight"
	WFHGreater3   = "Obese"

	// Categorizations for MUAC for Age
	MUACFAAgeLess6       = "Insufficient evidence to recommend a MUAC cutoff for children under 6 months of age"
	MUACFALess11p5CM     = "Severe Acute Malnutrition (SAM)"
	MUACFA11p5CMTo12p5CM = "Moderate Acute Malnutrition (MAM)"
	MUACGreater12p5CM    = "Normal (if other indicators indicate overweight/obese, they take precedence)"
)

// GetMUACFAResult returns a categorization for the subject's Mid Upper Arm Circumference for Age.
// Takes in a value for mid upper arm circumference and a age in months (0 to 60 inclusive)
// It returns a categorization and an error. Either the categorization or the error will be empty/nil,
// but not both. MUAC is in CM.
func (a *Anthropometry) GetMUACFAResult(muac float64, am int) (muacfa string, err error) {

	// Check parameters
	if err = checkAgeMonths(am); err != nil {
		return
	}

	switch {
	case am < 6:
		muacfa = MUACFAAgeLess6
	case muac < 11.5:
		muacfa = MUACFALess11p5CM
	case (muac >= 11.5) && (muac < 12.5):
		muacfa = MUACFA11p5CMTo12p5CM
	case muac > 12.5:
		muacfa = MUACGreater12p5CM
	default:
		err = fmt.Errorf("Invalid values for Mid Upper Arm Circumference And/Or Age in Months. MUAC: %f. Age in Months: %d\n",
			muac, am)
	}

	return
}

// GetWFHResult returns a categorization for the subject's Weight For Height.
// It takes in a weight, a gender (1 for boys, 0 for females),  a height
// in cm, and an age (0 to 60 inclusive) It returns a categorization and an error.
// Either the categorization or the error will be empty/nil, but not both.
func (a *Anthropometry) GetWFHResult(w float64, g int, h float64, am int) (wfh string, err error) {
	var zScore float64

	// Make sure the parameters are properly scoped
	if err = checkGender(g); err != nil {
		return
	}
	if err = checkAgeMonths(am); err != nil {
		return
	}
	if err = checkHeight(h, am); err != nil {
		return
	}

	if zScore, err = a.calcZScore(w, WFH, g, am, h); err != nil {
		return
	}
	switch {
	case zScore < -3:
		wfh = WFHLessNeg3
	case (zScore >= -3) && (zScore < -2):
		wfh = WFHNeg3ToNeg2
	case (zScore >= -2) && (zScore <= 1):
		wfh = WFHNeg2To1
	case (zScore > 1) && (zScore <= 2):
		wfh = WFH1To2
	case (zScore > 2) && (zScore <= 3):
		wfh = WFH2To3
	case (zScore > 3):
		wfh = WFHGreater3
	default:
		err = fmt.Errorf("Invalid zScore: %f\n", zScore)
	}

	return
}

// GetWFAResult returns a categorization for the subject's Weight For Age.
// It takes in a weight, a gender (1 for boys, 0 for females), and an age
// in months (0 to 60 inclusive). It returns a categorization and an error.
// Either the categorization or the error will be empty/nil, but not both.
func (a *Anthropometry) GetWFAResult(w float64, g int, am int) (wfa string, err error) {
	var zScore float64

	// Make sure the parameters are properly scoped
	if err = checkGender(g); err != nil {
		return
	}
	if err = checkAgeMonths(am); err != nil {
		return
	}

	// Pass in 100 for height just to satisfy signature. Height has no bearing on WFA zScore
	if zScore, err = a.calcZScore(w, WFA, g, am, 100); err != nil {
		return
	}
	switch {
	case zScore < -3:
		wfa = WFALessNeg3
	case (zScore >= -3) && (zScore < -2):
		wfa = WFANeg3ToNeg2
	case (zScore >= -2) && (zScore <= 1):
		wfa = WFANeg2To1
	case zScore > 1:
		wfa = WFAGreater1
	default:
		err = fmt.Errorf("Invalid zScore: %f\n", zScore)
	}

	return
}

// GetHFAResult returns a categorization for the subject's Hegiht For Age.
// It takes in a height, a gender (1 for boys, 0 for females), and an age
// in months (0 yo 60 inclusive). It returns a categorization and an error.
// Either the categorization or the error will be empty/nil, but not both.
func (a *Anthropometry) GetHFAResult(h float64, g int, am int) (hfa string, err error) {
	var zScore float64

	// Make sure the parameters are properly scoped
	if err = checkGender(g); err != nil {
		return
	}
	if err = checkAgeMonths(am); err != nil {
		return
	}

	// Pass in 100 for height just to satisfy signature. Height has no bearing on HFA zScore
	if zScore, err = a.calcZScore(h, HFA, g, am, 100); err != nil {
		return
	}
	switch {
	case zScore < -3:
		hfa = HFALessNeg3
	case (zScore >= -3) && (zScore < -2):
		hfa = HFANeg3ToNeg2
	case (zScore >= -2) && (zScore <= 3):
		hfa = HFANeg2To3
	case zScore > 3:
		hfa = HFAGreater3
	default:
		err = fmt.Errorf("Invalid zScore: %f\n", zScore)
	}

	return
}

// GetHCFAResult returns a categorization for the subject's head circumference for age.
// It takes in a head circumferece value, a gender (1 for boys, 0 for females), and
// an age in months (0 to 60 inclusve). It returns a categorization and an error.
// Either the categorization will be the empty string and the error will be non-nil,
// or the categorization will be a non-empty string and the error will be nil, but nevr both.
func (a *Anthropometry) GetHCFAResult(hc float64, g int, am int) (hcfa string, err error) {
	var zScore float64

	// Make sure the parameters are properly scoped
	if err = checkGender(g); err != nil {
		return
	}
	if err = checkAgeMonths(am); err != nil {
		return
	}

	// Pass in 100 for height just to satisfy signature. Height has no bearing on HCFA zScore
	if zScore, err = a.calcZScore(hc, HCFA, g, am, 100); err != nil {
		return
	}
	switch {
	case zScore < -3:
		hcfa = HCFALessNeg3
	case (zScore >= -3) && (zScore < -2):
		hcfa = HCFANeg3ToNeg2
	case (zScore >= -2) && (zScore <= 3):
		hcfa = HCFANeg2To3
	case zScore > 3:
		hcfa = HCFAGreater3
	default:
		err = fmt.Errorf("Invalid zScore: %f\n", zScore)
	}

	return
}

// calcZcore calculates the Z score for a subject given the data value, the name
// of the anthropometric indicator in question, the gender, the age in months, and the height.
func (a *Anthropometry) calcZScore(d float64, ain string, g int, am int, h float64) (zScoreRounded float64, err error) {
	var m, sd, zScoreRaw, s, l float64

	// Make sure the parameters are properly scoped
	if err = checkAIN(ain); err != nil {
		return
	}
	if err = checkGender(g); err != nil {
		return
	}
	if err = checkAgeMonths(am); err != nil {
		return
	}
	if err = checkHeight(h, am); err != nil {
		return
	}

	// Calculate Z scores
	a.checkAndInitWhoData()
	switch ain {
	case HCFA, HFA: // normally distributed data
		if m, err = globalWhoData.getValue(ain, g, am, h, "M"); err != nil {
			return
		}
		if sd, err = globalWhoData.getValue(ain, g, am, h, "SD"); err != nil {
			return
		}
		zScoreRaw = (d - m) / sd
	case WFA, WFH: // non normally distributed data
		if m, err = globalWhoData.getValue(ain, g, am, h, "M"); err != nil {
			return
		}
		if l, err = globalWhoData.getValue(ain, g, am, h, "L"); err != nil {
			return
		}
		if s, err = globalWhoData.getValue(ain, g, am, h, "S"); err != nil {
			return
		}
		zScoreRaw = (math.Pow((d/m), l) - 1) / (l * s)
	default:
		err = fmt.Errorf("invalid AIN (Anthropometric Indicator): %s\n", ain)
	}

	zScoreRounded = math.Round(zScoreRaw*10) / 10
	return
}

// checkAndInitWhoData checks to see if the global whodata variable is already initalized or not. If it
// isn't, it initalized it. If it is, it does nothing
func (a *Anthropometry) checkAndInitWhoData() {
	// If Environment parameter has its nil value, then whodata is uninitialized
	if globalWhoData.Config.Environment.App == "" {
		globalWhoData.Config = a.Config
	}
}

// Checks if AIN is one of the designated Anthropometric Indicators. If not, returns appropriate error
func checkAIN(ain string) (err error) {
	if (ain != HCFA) && (ain != HFA) && (ain != WFA) && (ain != WFH) {
		err = fmt.Errorf("invalid ain: %s\n", ain)
	}

	return
}

// Checks if gender is 1 (Female) or 2 (Boy) If not, returns an appropriate error.
func checkGender(g int) (err error) {
	if isGenderValid := ((g == 1) || (g == 2)); !isGenderValid {
		err = fmt.Errorf("gender must be 1 (for females) or 2 (for boys). Gender found: %d\n", g)
	}
	return nil
}

// Checks if age in months is between 0 and 60 exclusive. If not, returns an appropriate error
func checkAgeMonths(am int) (err error) {
	if isAgeMonthsValid := ((am >= 0) && (am <= 60)); !isAgeMonthsValid {
		err = fmt.Errorf("Age must be between 0 and 60 months. Current age: %d\n", am)
	}

	return
}

// Checks if length is between 45-110 for ages 0-24 months, or 65-120 for 2-5
func checkHeight(length float64, am int) (err error) {
	if (am <= 24) && ((length < 45) || (length > 110)) {
		err = fmt.Errorf("For age less than or equal to 24 months, length must be between 45 and 110. Age: %d, length: %f\n",
			am, length)
	} else if am > 24 && am <= 60 {
		if (length < 65) || (length > 120) {
			err = fmt.Errorf("For age between 24 (exclusive) and 60 months (inclusive), length must be between 45 and 110."+
				"Age: %d, length: %f\n", am, length)
		}
	}
	return
}

// Checks that either the age in months equals the age in the row , or that the
// height corresponds to the given row
func checkEquals(ain string, am int, length float64, row []string) (eq bool, err error) {
	var (
		rowAge    int
		rowLength float64
	)

	// Make sure we didn't get an empty row
	if len(row) == 0 {
		err = fmt.Errorf("Empty row retrieved from WHO database. Anthropometric indicator: %s, Age in Months: %d, length: %f\n",
			ain, am, length)
		return
	}

	// Check for equality
	switch ain {
	case HCFA, HFA, WFA:
		if rowAge, err = strconv.Atoi(row[0]); err != nil {
			return
		}
		// If the AIN in question is Head Circumference for age, Height for Age, or Weight for Age,
		// then we want to find the row in which our subject's Age in Months (am) equals the
		// row's Age in Months (rowAge).
		if rowAge == am {
			eq = true
		}
	case WFH:
		if rowLength, err = strconv.ParseFloat(row[0], 64); err != nil {
			return
		}
		// If the AIN in question is Weight for Height, then we want to find the row in which our
		// subject's length (length) is within 0.5 cm of the row length (rowLength).
		if (length - rowLength) < 0.5 {
			eq = true
		}
	default:
		err = fmt.Errorf("Invalid ain: %s\n", ain)
	}

	return
}
