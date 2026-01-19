package captcha

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"strings"

	vision "cloud.google.com/go/vision/v2/apiv1"
	"cloud.google.com/go/vision/v2/apiv1/visionpb"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

type GoogleCaptchaClient struct {
	client *vision.ImageAnnotatorClient
	ctx    context.Context
}

// NewGoogleCaptchaClient creates a new Google Vision OCR client
// credentialsJSON should be the JSON key file content as a string
func NewGoogleCaptchaClient(credentialsJSON string) (*GoogleCaptchaClient, error) {
	ctx := context.Background()

	var client *vision.ImageAnnotatorClient
	var err error

	if credentialsJSON != "" {
		// Use provided credentials
		creds, err := google.CredentialsFromJSON(ctx, []byte(credentialsJSON), "https://www.googleapis.com/auth/cloud-vision")
		if err != nil {
			return nil, err
		}
		client, err = vision.NewImageAnnotatorClient(ctx, option.WithCredentials(creds))
	} else {
		// Use default credentials (from GOOGLE_APPLICATION_CREDENTIALS env var)
		client, err = vision.NewImageAnnotatorClient(ctx)
	}

	if err != nil {
		return nil, err
	}

	return &GoogleCaptchaClient{
		client: client,
		ctx:    ctx,
	}, nil
}

func (g *GoogleCaptchaClient) DoWithBase64Img(base64Img string) (*CaptchaResponse, error) {
	// Remove data URL prefix if present
	base64Source := base64Img
	if idx := strings.Index(base64Img, ","); idx != -1 {
		base64Source = base64Img[idx+1:]
	}

	blob, err := base64.StdEncoding.DecodeString(base64Source)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(blob)
	return g.DoWithReader(buf)
}

func (g *GoogleCaptchaClient) DoWithReader(r io.Reader) (*CaptchaResponse, error) {
	// Read image data
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(r)
	if err != nil {
		return nil, err
	}

	// Create image from bytes
	image := &visionpb.Image{
		Content: buf.Bytes(),
	}

	// Create annotation request
	feature := &visionpb.Feature{
		Type: visionpb.Feature_TEXT_DETECTION,
	}

	request := &visionpb.AnnotateImageRequest{
		Image:    image,
		Features: []*visionpb.Feature{feature},
	}

	batchRequest := &visionpb.BatchAnnotateImagesRequest{
		Requests: []*visionpb.AnnotateImageRequest{request},
	}

	// Perform text detection
	response, err := g.client.BatchAnnotateImages(g.ctx, batchRequest)
	if err != nil {
		return nil, err
	}

	if len(response.Responses) == 0 || len(response.Responses[0].TextAnnotations) == 0 {
		return &CaptchaResponse{
			Content: "",
			Word:    "",
		}, nil
	}

	// The first annotation contains all detected text
	detectedText := response.Responses[0].TextAnnotations[0].Description

	return &CaptchaResponse{
		Content: detectedText,
		Word:    "",
	}, nil
}

// Close closes the Google Vision client
func (g *GoogleCaptchaClient) Close() error {
	return g.client.Close()
}
