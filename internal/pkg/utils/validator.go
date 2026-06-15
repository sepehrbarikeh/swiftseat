package utils


import (
    "github.com/go-playground/validator/v10"
)

var validate = validator.New()

type ErrorResponse struct {
    Field string `json:"field"`
    Tag   string `json:"tag"`
}

func ValidateStruct(s interface{}) []*ErrorResponse {
    var errors []*ErrorResponse
    err := validate.Struct(s)
    if err != nil {
        for _, err := range err.(validator.ValidationErrors) {
            errors = append(errors, &ErrorResponse{
                Field: err.Field(),
                Tag:   err.Tag(),
            })
        }
    }
    return errors
}