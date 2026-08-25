package responder

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/http/response"
	appI18n "CredChain_Golang/infrastructure/i18n"

	"github.com/gin-gonic/gin"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// Send writes a unified success or informational response.
func Send[T any](c *gin.Context, code int, data T) {
	localizer := appI18n.GetI18nLocalizer(c)
	msgKey := getMessageKey(code)
	message := localize(c, localizer, msgKey, nil)

	c.JSON(HttpCodeFromCode(code), response.Response[T]{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// SendPagination writes a unified paginated response, deriving page and limit
// from the query string.
func SendPagination[T any](c *gin.Context, code int, items []T, total int) {
	data := response.NewPaginationFromContext(c, items, total)
	Send(c, code, data)
}

// SendPaginationWithPageLimit writes a paginated response using the page and
// limit the handler actually applied. Use it wherever the handler defaults
// differ from the query string, so the envelope's limit and last_page match the
// rows returned.
func SendPaginationWithPageLimit[T any](c *gin.Context, code int, items []T, total, page, limit int) {
	data := response.NewPaginationWithPageLimit(c, items, total, page, limit)
	Send(c, code, data)
}

// SendError writes a unified error response and aborts the request.
func SendError(c *gin.Context, err error) {
	localizer := appI18n.GetI18nLocalizer(c)

	var appErr *domain.Error
	if errors.As(err, &appErr) {
		msgKey := getMessageKey(appErr.Code)
		message := localize(c, localizer, msgKey, buildTemplateData(appErr.Metadata))
		c.JSON(HttpCodeFromCode(appErr.Code), response.Response[any]{
			Code:    appErr.Code,
			Message: message,
		})
		c.Abort()
		return
	}

	if isMalformedBodyError(err) {
		msgKey := getMessageKey(domain.CodeSystemValidation)
		message := localize(c, localizer, msgKey, nil)
		c.JSON(HttpCodeFromCode(domain.CodeSystemValidation), response.Response[any]{
			Code:    domain.CodeSystemValidation,
			Message: message,
		})
		c.Abort()
		return
	}

	msgKey := getMessageKey(domain.CodeSystemInternal)
	message := localize(c, localizer, msgKey, nil)
	c.JSON(http.StatusInternalServerError, response.Response[any]{
		Code:    domain.CodeSystemInternal,
		Message: message,
	})
	c.Abort()
}

// isMalformedBodyError reports whether err indicates a malformed/empty/wrong-type
// request body that should produce 400 Bad Request rather than 500.
// Covers: io.EOF (empty body), io.ErrUnexpectedEOF (truncated body),
// *json.SyntaxError (non-JSON content), *json.UnmarshalTypeError (wrong field type).
func isMalformedBodyError(err error) bool {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return true
	}
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return true
	}
	return false
}

// SendValidationError handles ozzo-validation.Errors specifically.
func SendValidationError(c *gin.Context, err error) {
	localizer := appI18n.GetI18nLocalizer(c)

	vErrs, ok := err.(validation.Errors)
	if !ok {
		c.Error(err)
		SendError(c, domain.NewError(domain.CodeSystemInternal))
		return
	}

	fieldErrors := make(map[string][]string)
	for path, fieldErr := range vErrs {
		parts := strings.Split(path, ".")
		leaf := parts[len(parts)-1]

		label := leaf
		if translated := localize(c, localizer, "labels."+leaf, nil); translated != "" {
			label = translated
		}

		// Extract ozzo validation error details
		if vErr, ok := fieldErr.(validation.Error); ok {
			code := vErr.Code()
			params := vErr.Params()

			// Add field to params for i18n template
			if params == nil {
				params = make(map[string]interface{})
			}
			params["field"] = label

			// Use ozzo code directly as i18n key
			message := localize(c, localizer, code, params)
			if message == "" {
				message = vErr.Message()
			}
			fieldErrors[path] = append(fieldErrors[path], message)
		} else {
			// Fallback for non-ozzo errors
			fieldErrors[path] = append(fieldErrors[path], fieldErr.Error())
		}
	}

	msgKey := getMessageKey(domain.CodeSystemValidation)
	message := localize(c, localizer, msgKey, nil)

	c.JSON(HttpCodeFromCode(domain.CodeSystemValidation), response.Response[any]{
		Code:    domain.CodeSystemValidation,
		Message: message,
		Errors:  fieldErrors,
	})
	c.Abort()
}

func getMessageKey(code int) string {
	msgKey, keyOk := CodeToMessageKey[code]
	if !keyOk {
		return fmt.Sprintf("unregistered_code_%d", code)
	}
	return msgKey
}

func buildTemplateData(metadata map[string]any) map[string]any {
	if metadata == nil {
		return nil
	}
	data := make(map[string]any)
	for k, v := range metadata {
		switch val := v.(type) {
		case []string:
			if len(val) == 0 {
				data[k] = "(none)"
			} else {
				data[k] = strings.Join(val, ", ")
			}
		case []int:
			if len(val) == 0 {
				data[k] = "(none)"
			} else {
				strs := make([]string, len(val))
				for i, v := range val {
					strs[i] = strconv.Itoa(v)
				}
				data[k] = strings.Join(strs, ", ")
			}
		case []int64:
			if len(val) == 0 {
				data[k] = "(none)"
			} else {
				strs := make([]string, len(val))
				for i, v := range val {
					strs[i] = strconv.FormatInt(v, 10)
				}
				data[k] = strings.Join(strs, ", ")
			}
		case []uint8:
			if len(val) == 0 {
				data[k] = "(none)"
			} else {
				strs := make([]string, len(val))
				for i, v := range val {
					strs[i] = strconv.FormatUint(uint64(v), 10)
				}
				data[k] = strings.Join(strs, ", ")
			}
		case []bool:
			if len(val) == 0 {
				data[k] = "(none)"
			} else {
				strs := make([]string, len(val))
				for i, v := range val {
					strs[i] = strconv.FormatBool(v)
				}
				data[k] = strings.Join(strs, ", ")
			}
		default:
			data[k] = v
		}
	}
	return data
}

func localize(c *gin.Context, localizer *i18n.Localizer, msgID string, data map[string]any) string {
	if msgID == "" {
		return ""
	}
	if localizer == nil {
		c.Error(fmt.Errorf("responder.localize: i18n localizer is nil (msgID=%q)", msgID)) //nolint:errcheck
		return msgID
	}

	text, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    msgID,
		TemplateData: data,
	})
	if err != nil {
		c.Error(fmt.Errorf("responder.localize: i18n miss msgID=%q: %w", msgID, err)) //nolint:errcheck
		return msgID
	}
	return text
}
func resolveMessage(c *gin.Context, code int) string {
	localizer := appI18n.GetI18nLocalizer(c)
	msgKey := getMessageKey(code)
	return localize(c, localizer, msgKey, nil)
}

// ResolveMessage returns the localized message for a code using the request's localizer.
func ResolveMessage(c *gin.Context, code int) string {
	return resolveMessage(c, code)
}
