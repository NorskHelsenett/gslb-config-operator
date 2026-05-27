package models

type G3Record struct {
	Id             *int         `json:"id,omitempty"`
	Name           string       `json:"name"`
	Address        string       `json:"address"`
	Aux            *string      `json:"aux,omitempty"`
	Comment        *string      `json:"comment,omitempty"`
	Created_At     *string      `json:"created_at,omitempty"`
	Updated_At     *string      `json:"updated_at,omitempty"`
	Dns_Zone_Id    *int         `json:"dns_zone_id,omitempty"`
	Ttl            string       `json:"ttl,omitempty"`
	Type           G3RecordType `json:"type"`
	Infrastructure string       `json:"infrastructure"`
}

type G3PaginatedResponse struct {
	Current_Page  int        `json:"current_page"`
	Per_Page      int        `json:"per_page"`
	Total_Entries int        `json:"total_entries"`
	Items         []G3Record `json:"items"`
}

type G3MutationResponse struct {
	Msg string `json:"msg"`
}

type G3Zone struct {
	Id             int    `json:"id"`
	Name           string `json:"name"`
	Name_Reverse   string `json:"name_reverse"`
	Location_Id    int    `json:"location_id"`
	Status_Id      int    `json:"status_id"`
	Infrastructure string `json:"infrastructure"`
	Description    string `json:"description"`
	Dirty          bool   `json:"dirty"`
	Created_At     string `json:"created_at"`
	Updated_At     string `json:"updated_at"`
	Default_Ttl    int    `json:"default_ttl"`
	Publish        bool   `json:"publish"`
}
