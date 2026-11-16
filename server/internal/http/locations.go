package httpapi

import (
	"strconv"
	"strings"
)

type TownCoordinate struct {
	Name string
	Lat  float64
	Lng  float64
}

var KnownTowns = map[string]TownCoordinate{
	"ratnapura":     {Name: "Ratnapura", Lat: 6.6828, Lng: 80.3992},
	"mirissa":       {Name: "Mirissa", Lat: 5.9483, Lng: 80.4552},
	"weligama":      {Name: "Weligama", Lat: 5.9723, Lng: 80.4287},
	"galle":         {Name: "Galle", Lat: 6.0535, Lng: 80.2210},
	"galle fort":    {Name: "Galle Fort", Lat: 6.0535, Lng: 80.2210},
	"colombo":       {Name: "Colombo", Lat: 6.9271, Lng: 79.8612},
	"ella":          {Name: "Ella", Lat: 6.8667, Lng: 81.0466},
	"kandy":         {Name: "Kandy", Lat: 7.2906, Lng: 80.6337},
	"sigiriya":      {Name: "Sigiriya", Lat: 7.9570, Lng: 80.7603},
	"arugam bay":    {Name: "Arugam Bay", Lat: 6.8417, Lng: 81.8358},
	"arugambay":     {Name: "Arugam Bay", Lat: 6.8417, Lng: 81.8358},
	"nuwara eliya":  {Name: "Nuwara Eliya", Lat: 6.9497, Lng: 80.7891},
	"nuwaraeliya":   {Name: "Nuwara Eliya", Lat: 6.9497, Lng: 80.7891},
	"negombo":       {Name: "Negombo", Lat: 7.2008, Lng: 79.8737},
	"bentota":       {Name: "Bentota", Lat: 6.4252, Lng: 79.9984},
	"tangalle":      {Name: "Tangalle", Lat: 6.0242, Lng: 80.7937},
	"hiriketiya":    {Name: "Hiriketiya", Lat: 5.9647, Lng: 80.7061},
	"dambulla":      {Name: "Dambulla", Lat: 7.8742, Lng: 80.6511},
	"jaffna":        {Name: "Jaffna", Lat: 9.6615, Lng: 80.0255},
	"trincomalee":   {Name: "Trincomalee", Lat: 8.5874, Lng: 81.2152},
	"batticaloa":    {Name: "Batticaloa", Lat: 7.7310, Lng: 81.6747},
	"anuradhapura":  {Name: "Anuradhapura", Lat: 8.3114, Lng: 80.4037},
	"polonnaruwa":   {Name: "Polonnaruwa", Lat: 7.9403, Lng: 81.0188},
	"matara":        {Name: "Matara", Lat: 5.9549, Lng: 80.5550},
	"badulla":       {Name: "Badulla", Lat: 6.9934, Lng: 81.0550},
	"hikkaduwa":     {Name: "Hikkaduwa", Lat: 6.1394, Lng: 80.1063},
	"unawatuna":     {Name: "Unawatuna", Lat: 6.0174, Lng: 80.2483},
	"pasikuda":      {Name: "Pasikuda", Lat: 7.9272, Lng: 81.5645},
	"yala":          {Name: "Yala", Lat: 6.2828, Lng: 81.2872},
	"udawalawe":     {Name: "Udawalawe", Lat: 6.4428, Lng: 80.8872},
	"kalutara":      {Name: "Kalutara", Lat: 6.5854, Lng: 79.9607},
	"mount lavinia": {Name: "Mount Lavinia", Lat: 6.8344, Lng: 79.8654},
}

func resolveTownCoordinates(input string) (*float64, *float64) {
	s := strings.TrimSpace(strings.ToLower(input))
	if s == "" {
		return nil, nil
	}
	if town, ok := KnownTowns[s]; ok {
		lat, lng := town.Lat, town.Lng
		return &lat, &lng
	}
	parts := strings.Split(s, ",")
	if len(parts) == 2 {
		latVal, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		lngVal, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err1 == nil && err2 == nil {
			return &latVal, &lngVal
		}
	}
	return nil, nil
}
