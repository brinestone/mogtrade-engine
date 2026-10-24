package mesomb

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/brinestone/mogtrade/core/contract"
)

type MesombPaymentProcessor struct {
	client         *http.Client
	keyProvider    func() MesombKeys
	appId          string
	defaultTimeout time.Duration
}

func MakeDepositParams(params DepositParams) contract.InitializeDepositParams {
	return params
}

func (m *MesombPaymentProcessor) InitializeDeposit(ctx context.Context, p contract.InitializeDepositParams) (contract.PaymentIntent, error) {
	nonce := generateNonce()
	parsed := p.(DepositParams)
	amount, _ := parsed.Amount.Float64()
	res, err := m.collectFunds(ctx, collectFundsParams{
		TransactionId: parsed.TransactionId,
		Date:          time.Now(),
		Nonce:         nonce,
		Body: map[string]any{
			"payer":    parsed.LocalPhoneNumber,
			"amount":   amount,
			"service":  parsed.Service,
			"country":  parsed.Country,
			"currency": parsed.Currency,
			"trxID":    parsed.TransactionId,
			"message":  parsed.Note,
		},
	})
	if err != nil {
		return contract.PaymentIntent{}, err
	}
	return contract.PaymentIntent{
		ProviderRef: res.Transaction.Pk,
		ID:          parsed.TransactionId,
		Provider:    "mesomb",
		Amount:      parsed.Amount,
		Currency:    res.Transaction.Currency,
		Status:      contract.PaymentStatus(strings.ToLower(res.Transaction.Status)),
	}, nil
}

func NewMesombPaymentProcessor(appId string, keyProvider func() MesombKeys) *MesombPaymentProcessor {
	client := http.DefaultClient
	return &MesombPaymentProcessor{
		appId:          appId,
		client:         client,
		keyProvider:    keyProvider,
		defaultTimeout: time.Second * 30,
	}
}

func (m *MesombPaymentProcessor) collectFunds(ctx context.Context, params collectFundsParams) (mesombCollectResponse, error) {
	if params.Body == nil {
		params.Body = make(map[string]any)
	}

	if params.Mode == "" {
		params.Mode = "asynchronous"
	}

	if params.Timeout == 0 {
		params.Timeout = m.defaultTimeout
	}
	if params.Nonce == "" {
		params.Nonce = generateNonce()
	}
	if params.Language == "" {
		params.Language = "en"
	}

	u := buildUrl("/payment/collect")
	headers := map[string]string{
		"x-mesomb-date":          fmt.Sprintf("%d", params.Date.UnixMilli()),
		"x-mesomb-nonce":         params.Nonce,
		"Content-Type":           "application/json",
		"X-MeSomb-OperationMode": params.Mode,
		"Accept-Language":        params.Language,
		"X-MeSomb-Application":   m.appId,
	}

	if params.TransactionId != "" {
		headers["X-MeSomb-TrxID"] = params.TransactionId
	}

	authorization := buildAuthorization("POST", "/payment/collect", params.Nonce, "payment", params.Date, map[string]string{
		"content-type": "application/json",
	}, params.Body, m.keyProvider())

	headers["Authorization"] = authorization

	serialized, err := json.Marshal(params.Body)
	if err != nil {
		return mesombCollectResponse{}, err
	}
	body := bytes.NewReader(serialized)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, body)
	if err != nil {
		return mesombCollectResponse{}, err
	}

	res, err := m.client.Do(req)
	if err != nil {
		return mesombCollectResponse{}, err
	}
	defer res.Body.Close()
	rawBody, err := io.ReadAll(res.Body)

	if err != nil {
		return mesombCollectResponse{}, err
	}

	if res.StatusCode >= 400 {
		var errPayload mesombErrorResponse
		err = json.Unmarshal(rawBody, &errPayload)
		if err != nil {
			return mesombCollectResponse{}, err
		}
		return mesombCollectResponse{}, fmt.Errorf("%w: %s", ErrMesombRequestFailed, errPayload.Detail)
	} else {
		var response mesombCollectResponse
		err = json.Unmarshal(rawBody, &response)
		if err != nil {
			return mesombCollectResponse{}, err
		}
		return response, nil
	}
}

func buildAuthorization(method, endpoint, nonce, service string, date time.Time, headers map[string]string, body any, credentials MesombKeys) string {
	url := buildUrl(endpoint)
	return signRequest(signRequestParams{
		Algorithm:   algorithm,
		Service:     service,
		Method:      method,
		Url:         url,
		Date:        mesombTimestamp(date),
		Nonce:       nonce,
		Credentials: credentials,
		Headers:     headers,
		Body:        body,
	})
}

func buildCanonicalRequestData(params signRequestParams) canonicalRequestData {
	parsed := parseUrl(params.Url)
	timestamp := params.Date
	canonicalQuery := parsed.Query
	headers := params.Headers

	headers["host"] = fmt.Sprintf("%s//%s", parsed.Protocol, parsed.Host)
	headers["x-mesomb-date"] = timestamp.String()
	headers["x-mesomb-nonce"] = params.Nonce

	headerKeys := make([]string, len(headers))
	for key := range maps.Keys(headers) {
		headerKeys = append(headerKeys, key)
	}

	slices.Sort(headerKeys)

	var canonicalHeaders []string
	for _, headerKey := range headerKeys {
		canonicalHeaders = append(canonicalHeaders, fmt.Sprintf("%s:%s", headerKey, headers[headerKey]))
	}
	canonicalHeaderString := strings.Join(canonicalHeaders, "\n")
	signedHeaders := strings.Join(headerKeys, ";")
	path := url.PathEscape(parsed.Path)
	scope := fmt.Sprintf("%d%d%d/%s/mesomb_request", time.Time(params.Date).Year(), time.Time(params.Date).Month(), time.Time(params.Date).Day(), params.Service)

	return canonicalRequestData{
		Timestamp:        timestamp,
		Query:            canonicalQuery,
		Path:             path,
		HostHeader:       headers["host"],
		Headers:          headers,
		HeaderKeys:       headerKeys,
		CanonicalHeaders: canonicalHeaderString,
		SignedHeaders:    signedHeaders,
		Scope:            scope,
	}
}

func parseUrl(raw string) parsedUrl {
	u, _ := url.Parse(raw)
	return parsedUrl{
		Protocol: u.Scheme,
		Host:     u.Hostname(),
		Path:     u.Path,
		Query:    u.RawQuery,
	}
}

func buildUrl(endpoint string) string {
	return fmt.Sprintf("%s/%s", baseUrl, endpoint)
}

func buildCanonicalRequest(method, path, canonicalQuery, canonicalHeaders, signedHeaders, payloadHash string) string {
	return strings.Join([]string{
		method, path, canonicalQuery, canonicalHeaders, signedHeaders, payloadHash,
	}, "\n")
}

func generateNonce() string {
	buf := make([]byte, 20)
	rand.Reader.Read(buf)
	return base64.RawStdEncoding.EncodeToString(buf)
}

func signRequest(params signRequestParams) string {
	req := buildCanonicalRequestData(params)

	cipher := sha1.New()

	serialized, _ := json.Marshal(params.Body)
	cipher.Write(serialized)
	payloadHash := string(cipher.Sum([]byte{}))
	cReq := buildCanonicalRequest(params.Method, req.Path, req.Query, req.CanonicalHeaders, req.SignedHeaders, payloadHash)

	cipher = sha1.New()
	cipher.Write([]byte(cReq))
	stringToSign := strings.Join([]string{
		params.Algorithm,
		req.Timestamp.String(),
		req.Scope,
		string(cipher.Sum([]byte{})),
	}, "\n")

	cipher = hmac.New(sha1.New, []byte(params.Credentials.Secret))
	signature := cipher.Sum([]byte(stringToSign))

	return fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s", params.Algorithm, params.Credentials.Access, req.Scope, req.SignedHeaders, signature)
}
