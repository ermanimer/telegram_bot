package telegram_bot

type input struct {
	Start   bool
	Stop    bool
	Message string
}

type output struct {
	InfoMessage  string
	ErrorMessage string
}

type getUpdatesRequest struct {
	Offset int `json:"offset"`
}

type getUpdatesResponse struct {
	Ok     bool `json:"ok"`
	Result []struct {
		UpdateId int `json:"update_id"`
		Message  struct {
			MessageId int `json:"message_id"`
			From      struct {
				Id           int    `json:"id"`
				IsBot        bool   `json:"is_bot"`
				FirstName    string `json:"first_name"`
				LastName     string `json:"last_name"`
				LanguageCode string `json:"language_code"`
			} `json:"from"`
			Chat struct {
				Id        int    `json:"id"`
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
				Type      string `json:"type"`
			} `json:"chat"`
			Date     int    `json:"date"`
			Text     string `json:"text"`
			Entities []struct {
				Offset int    `json:"offset"`
				Length int    `json:"length"`
				Type   string `json:"type"`
			} `json:"entities"`
		} `json:"message"`
	} `json:"result"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}

type sendMessageRequest struct {
	ChatId int    `json:"chat_id"`
	Text   string `json:"text"`
}

type sendMessageResponse struct {
	Ok     bool `json:"ok"`
	Result struct {
		MessageId int `json:"message_id"`
		From      struct {
			Id        int    `json:"id"`
			IsBot     bool   `json:"is_bot"`
			FirstName string `json:"first_name"`
			Username  string `json:"username"`
		} `json:"from"`
		Chat struct {
			Id        int    `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Type      string `json:"type"`
		} `json:"chat"`
		Date int    `json:"date"`
		Text string `json:"text"`
	} `json:"result"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}
