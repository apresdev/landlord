package webapp

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"strings"

	"github.com/apresdev/landlord/internal/rest"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

//go:embed shared-templates/*
var sharedTemplatesFS embed.FS

//go:embed login-templates/*
var loginTemplatesFS embed.FS

//go:embed admin-templates/*
var adminTemplatesFS embed.FS

func readDir(fs embed.FS, dir string) ([]fs.DirEntry, error) {
	f, err := fs.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	return f, nil
}

type FileResult struct {
	Name  string
	Bytes []byte
}

func (fr FileResult) Contents() string {
	return string(fr.Bytes)
}

func readAllFilesInDir(fs embed.FS, dir string) ([]FileResult, error) {
	f, err := readDir(fs, dir)
	if err != nil {
		return nil, err
	}
	results := make([]FileResult, 0, len(f))
	for _, entry := range f {
		b, err := fs.ReadFile(fmt.Sprintf("%s/%s", dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		results = append(results, FileResult{Name: entry.Name(), Bytes: b})
	}
	return results, nil
}

var loginTemplates *template.Template
var adminTemplates *template.Template

func parseTemplates(t *template.Template, fs embed.FS, subDir string) (*template.Template, error) {
	theFiles, err := readAllFilesInDir(fs, subDir)
	if err != nil {
		return nil, err
	}
	for _, file := range theFiles {
		t, err = t.Parse(file.Contents())
		if err != nil {
			return nil, apperr.NewBadRequestE(
				fmt.Errorf("could not parse '%s': %w", file.Name, err))
		}
	}
	return t, nil
}

func init() {
	var err error
	loginTemplates = template.New("login-templates")
	adminTemplates = template.New("admin-templates")
	funcs := template.FuncMap{
		"strIf": func(lhs, rhs, result string) string {
			if lhs == rhs {
				return result
			}
			return ""
		},
		"ifElse": func(v bool, trueResult, falseResult string) string {
			if v {
				return trueResult
			}
			return falseResult
		},
		"isImpersonating": func(sub SubjectView, user UserView) bool {
			return sub.IsImpersonatingUser(user)
		},
		"isUser": func(sub SubjectView, user UserView) bool {
			return sub.ID() == user.ID()
		},
		"stringsJoin": func(s []string, sep string) string {
			return strings.Join(s, sep)
		},
	}
	loginTemplates.Funcs(funcs)
	loginTemplates, err = parseTemplates(loginTemplates, sharedTemplatesFS, "shared-templates")
	if err != nil {
		zap.L().Fatal("error parsing shared template files", zap.Error(err))
	}
	loginTemplates, err = parseTemplates(loginTemplates, loginTemplatesFS, "login-templates")
	if err != nil {
		zap.L().Fatal("error parsing login template files", zap.Error(err))
	}

	adminTemplates.Funcs(funcs)
	adminTemplates, err = parseTemplates(adminTemplates, sharedTemplatesFS, "shared-templates")
	if err != nil {
		zap.L().Fatal("error parsing shared template files", zap.Error(err))
	}
	adminTemplates, err = parseTemplates(adminTemplates, adminTemplatesFS, "admin-templates")
	if err != nil {
		zap.L().Fatal("error parsing admin template files", zap.Error(err))
	}
}

func executeUnauthenticatedView(c *gin.Context, name string, uv UnauthenticatedView) error {
	return loginTemplates.ExecuteTemplate(c.Writer, name, uv)
}

func executeFragmentTemplate(c *gin.Context, name string, fv FragmentView) error {
	return adminTemplates.ExecuteTemplate(c.Writer, name, fv)
}

func executeAdminTemplate(c *gin.Context, name string, asv AuthenticatedSubjectView) error {
	return adminTemplates.ExecuteTemplate(c.Writer, name, asv)
}

func executeTenantTemplate(c *gin.Context, name string, tv TenantView) error {
	return adminTemplates.ExecuteTemplate(c.Writer, name, tv)
}

func executeUserTemplate(c *gin.Context, name string, uv *UserView) error {
	return adminTemplates.ExecuteTemplate(c.Writer, name, uv)
}

func executeProfileFieldTemplate(c *gin.Context, name string, pfv ProfileFieldView) error {
	return adminTemplates.ExecuteTemplate(c.Writer, name, pfv)
}

func executeErrorTemplate(c *gin.Context, name string, epv ErrorPageView) error {
	return adminTemplates.ExecuteTemplate(c.Writer, name, epv)
}

func executeStringTemplate(c *gin.Context, name string, s string) error {
	return adminTemplates.ExecuteTemplate(c.Writer, name, s)
}

func executeUUIDTemplate(c *gin.Context, name string, key string) error {
	tenantID, err := rest.UUIDParam(c, key)
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	return adminTemplates.ExecuteTemplate(c.Writer, name, tenantID.String())
}
