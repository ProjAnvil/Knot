package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ProjAnvil/knot/backend/internal/models"
)

// BuildParameterTree builds a hierarchical tree structure from flat parameter list
func BuildParameterTree(params []models.Parameter) []models.Parameter {
	if len(params) == 0 {
		return []models.Parameter{}
	}

	// Create a map for quick lookup - store pointers to work with references
	paramMap := make(map[uint]*models.Parameter)

	// Make a copy of parameters to avoid modifying the original slice
	paramsCopy := make([]models.Parameter, len(params))
	for i := range params {
		paramsCopy[i] = params[i]
		paramsCopy[i].Children = []models.Parameter{}
		paramMap[paramsCopy[i].ID] = &paramsCopy[i]
	}

	// Build tree structure
	var roots []models.Parameter
	for i := range paramsCopy {
		if paramsCopy[i].ParentID == nil {
			// This is a root node, we'll add it later after building the tree
		} else {
			parent := paramMap[*paramsCopy[i].ParentID]
			if parent != nil {
				parent.Children = append(parent.Children, paramsCopy[i])
			}
		}
	}

	// Collect root nodes with their fully built children
	for i := range paramsCopy {
		if paramsCopy[i].ParentID == nil {
			roots = append(roots, *paramMap[paramsCopy[i].ID])
		}
	}

	return roots
}

// GenerateParameterHTML generates HTML table for parameters
func GenerateParameterHTML(params []models.Parameter, depth int) string {
	if len(params) == 0 {
		return `<p class="text-muted">No parameters</p>`
	}

	var html strings.Builder
	if depth == 0 {
		html.WriteString(`<table class="param-table"><thead><tr><th>Name</th><th>Type</th><th>Required</th><th>Description</th></tr></thead><tbody>`)
	}

	for _, param := range params {
		indent := strings.Repeat("&nbsp;", depth*4)
		prefix := ""
		if depth > 0 {
			prefix = "└─ "
		}

		required := `<span class="badge-optional">Optional</span>`
		if param.Required {
			required = `<span class="badge-required">Required</span>`
		}

		description := "-"
		if param.Description != nil && *param.Description != "" {
			description = *param.Description
		}

		html.WriteString(fmt.Sprintf(`<tr>
      <td class="param-name">%s%s%s</td>
      <td><span class="type-badge type-%s">%s</span></td>
      <td>%s</td>
      <td>%s</td>
    </tr>`, indent, prefix, param.Name, param.Type, param.Type, required, description))

		if len(param.Children) > 0 {
			html.WriteString(GenerateParameterHTML(param.Children, depth+1))
		}
	}

	if depth == 0 {
		html.WriteString(`</tbody></table>`)
	}
	return html.String()
}

// GenerateExampleJSON generates example JSON from parameters
func GenerateExampleJSON(params []models.Parameter) map[string]interface{} {
	result := make(map[string]interface{})

	for _, param := range params {
		var value interface{}
		switch param.Type {
		case "string":
			if param.Description != nil && *param.Description != "" {
				value = *param.Description
			} else {
				value = "string"
			}
		case "number":
			value = 0
		case "boolean":
			value = false
		case "array":
			if len(param.Children) > 0 {
				// Check if all children are primitive types
				allPrimitive := true
				for _, child := range param.Children {
					if child.Type == "object" || child.Type == "array" {
						allPrimitive = false
						break
					}
				}

				if allPrimitive && len(param.Children) == 1 {
					child := param.Children[0]
					var primitiveValue interface{}
					switch child.Type {
					case "string":
						if child.Description != nil && *child.Description != "" {
							primitiveValue = *child.Description
						} else {
							primitiveValue = "string"
						}
					case "number":
						primitiveValue = 0
					case "boolean":
						primitiveValue = false
					default:
						primitiveValue = nil
					}
					value = []interface{}{primitiveValue}
				} else {
					// Complex array with objects or multiple items
					value = []interface{}{GenerateExampleJSON(param.Children)}
				}
			} else {
				value = []interface{}{}
			}
		case "object":
			if len(param.Children) > 0 {
				value = GenerateExampleJSON(param.Children)
			} else {
				value = map[string]interface{}{}
			}
		default:
			value = nil
		}

		result[param.Name] = value
	}

	return result
}

// APIWithParams represents an API with its parameters separated by type
type APIWithParams struct {
	API                models.API
	GroupName          string
	RequestParameters  []models.Parameter
	ResponseParameters []models.Parameter
}

// GenerateHTML generates a complete HTML document from APIs
func GenerateHTML(apis []APIWithParams, locale string) string {
	title := "API Documentation"
	generatedAt := "Generated at"
	requestParams := "Request Parameters"
	responseParams := "Response Parameters"
	requestExample := "Request Example"
	responseExample := "Response Example"
	tableOfContents := "Table of Contents"

	if locale == "zh" {
		title = "API 文档"
		generatedAt = "生成时间"
		requestParams = "请求参数"
		responseParams = "响应参数"
		requestExample = "请求示例"
		responseExample = "响应示例"
		tableOfContents = "目录"
	}

	// Group APIs by group
	type GroupedAPIs struct {
		GroupID   uint
		GroupName string
		APIs      []APIWithParams
	}

	groupMap := make(map[uint]*GroupedAPIs)
	for _, api := range apis {
		groupID := api.API.GroupID
		if _, exists := groupMap[groupID]; !exists {
			groupMap[groupID] = &GroupedAPIs{
				GroupID:   groupID,
				GroupName: api.GroupName,
				APIs:      []APIWithParams{},
			}
		}
		groupMap[groupID].APIs = append(groupMap[groupID].APIs, api)
	}

	// Convert map to slice
	var groupedAPIs []*GroupedAPIs
	for _, group := range groupMap {
		groupedAPIs = append(groupedAPIs, group)
	}

	// Generate sidebar
	var sidebarItems strings.Builder
	for _, group := range groupedAPIs {
		sidebarItems.WriteString(`<div class="sidebar-group">`)
		sidebarItems.WriteString(fmt.Sprintf(`<div class="sidebar-group-title">%s</div>`, group.GroupName))
		sidebarItems.WriteString(`<div class="sidebar-group-items">`)

		for j, api := range group.APIs {
			globalIndex := 0
			for i, a := range apis {
				if a.API.ID == api.API.ID {
					globalIndex = i
					break
				}
			}
			_ = j
			sidebarItems.WriteString(fmt.Sprintf(`<a href="#api-%d" class="sidebar-api-item">%s</a>`, globalIndex, api.API.Name))
		}

		sidebarItems.WriteString(`</div></div>`)
	}

	// Generate API sections
	var apisHTML strings.Builder
	for index, api := range apis {
		requestTree := BuildParameterTree(api.RequestParameters)
		responseTree := BuildParameterTree(api.ResponseParameters)
		requestJSON := GenerateExampleJSON(requestTree)
		responseJSON := GenerateExampleJSON(responseTree)

		requestJSONBytes, _ := json.MarshalIndent(requestJSON, "", "  ")
		responseJSONBytes, _ := json.MarshalIndent(responseJSON, "", "  ")

		apisHTML.WriteString(fmt.Sprintf(`
      <div class="api-section" id="api-%d">
        <div class="api-header">
          <h2>%s</h2>
          <div class="api-meta">
            <span class="badge badge-%s">%s</span>
            <code class="endpoint">%s</code>
            <span class="badge badge-type">%s</span>
          </div>
        </div>

        <div class="section">
          <h3>%s</h3>
          %s
        </div>
`, index, api.API.Name, strings.ToLower(api.API.Method), api.API.Method, api.API.Endpoint, api.API.Type, requestParams, GenerateParameterHTML(requestTree, 0)))

		if len(requestJSON) > 0 {
			apisHTML.WriteString(fmt.Sprintf(`
        <div class="section">
          <h3>%s</h3>
          <pre><code>%s</code></pre>
        </div>
`, requestExample, string(requestJSONBytes)))
		}

		apisHTML.WriteString(fmt.Sprintf(`
        <div class="section">
          <h3>%s</h3>
          %s
        </div>
`, responseParams, GenerateParameterHTML(responseTree, 0)))

		if len(responseJSON) > 0 {
			apisHTML.WriteString(fmt.Sprintf(`
        <div class="section">
          <h3>%s</h3>
          <pre><code>%s</code></pre>
        </div>
`, responseExample, string(responseJSONBytes)))
		}

		apisHTML.WriteString(`      </div>`)
	}

	// Generate full HTML
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="%s">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
      line-height: 1.6;
      color: #333;
      background: #f5f5f5;
      display: flex;
      min-height: 100vh;
    }
    .sidebar {
      width: 280px;
      background: white;
      border-right: 1px solid #e5e5e5;
      padding: 20px;
      position: fixed;
      height: 100vh;
      overflow-y: auto;
      left: 0;
      top: 0;
    }
    .sidebar h1 {
      font-size: 1.5em;
      margin-bottom: 10px;
      color: #1a1a1a;
      border-bottom: 2px solid #0070f3;
      padding-bottom: 10px;
    }
    .sidebar-meta {
      font-size: 0.75em;
      color: #666;
      margin-bottom: 20px;
    }
    .sidebar-group {
      margin-bottom: 20px;
    }
    .sidebar-group-title {
      font-size: 0.85em;
      font-weight: 700;
      color: #1a1a1a;
      text-transform: uppercase;
      letter-spacing: 0.5px;
      padding: 8px 12px;
      margin-bottom: 8px;
      background: #f8f9fa;
      border-radius: 6px;
      border-left: 3px solid #0070f3;
    }
    .sidebar-api-item {
      display: block;
      padding: 8px 12px;
      color: #333;
      text-decoration: none;
      border-radius: 6px;
      margin-bottom: 4px;
      transition: all 0.2s;
      font-size: 0.9em;
    }
    .sidebar-api-item:hover {
      background: #f0f0f0;
      color: #0070f3;
    }
    .sidebar-api-item.active {
      background: #e3f2fd;
      color: #0070f3;
      font-weight: 500;
    }
    .main-content {
      flex: 1;
      margin-left: 280px;
      padding: 40px;
      max-width: calc(100%% - 280px);
    }
    .container {
      max-width: 1000px;
      margin: 0 auto;
      background: white;
      padding: 40px;
      border-radius: 8px;
      box-shadow: 0 2px 4px rgba(0,0,0,0.1);
    }
    .api-section {
      margin-bottom: 60px;
      padding-bottom: 40px;
      border-bottom: 2px solid #e5e5e5;
      scroll-margin-top: 20px;
    }
    .api-section:last-child {
      border-bottom: none;
    }
    .api-header h2 {
      font-size: 2em;
      margin-bottom: 15px;
      color: #1a1a1a;
    }
    .api-meta {
      display: flex;
      gap: 12px;
      align-items: center;
      margin-bottom: 30px;
      flex-wrap: wrap;
    }
    .badge {
      display: inline-block;
      padding: 4px 12px;
      border-radius: 4px;
      font-size: 0.85em;
      font-weight: 600;
      text-transform: uppercase;
    }
    .badge-get { background: #e3f2fd; color: #1976d2; }
    .badge-post { background: #e8f5e9; color: #388e3c; }
    .badge-put { background: #fff3e0; color: #f57c00; }
    .badge-delete { background: #ffebee; color: #d32f2f; }
    .badge-patch { background: #f3e5f5; color: #7b1fa2; }
    .badge-type { background: #f5f5f5; color: #666; }
    .endpoint {
      background: #f5f5f5;
      padding: 6px 12px;
      border-radius: 4px;
      font-family: 'Monaco', 'Menlo', monospace;
      font-size: 0.9em;
      color: #d63384;
    }
    .section {
      margin-bottom: 30px;
    }
    .section h3 {
      font-size: 1.4em;
      margin-bottom: 15px;
      color: #333;
      border-left: 4px solid #0070f3;
      padding-left: 12px;
    }
    .param-table {
      width: 100%%;
      border-collapse: collapse;
      margin-bottom: 20px;
      font-size: 0.95em;
    }
    .param-table th {
      background: #f8f9fa;
      padding: 12px;
      text-align: left;
      font-weight: 600;
      border-bottom: 2px solid #dee2e6;
      color: #495057;
    }
    .param-table td {
      padding: 12px;
      border-bottom: 1px solid #e9ecef;
      vertical-align: top;
    }
    .param-table tr:hover {
      background: #f8f9fa;
    }
    .param-name {
      font-family: 'Monaco', 'Menlo', monospace;
      font-weight: 500;
      color: #0066cc;
    }
    .type-badge {
      display: inline-block;
      padding: 2px 8px;
      border-radius: 3px;
      font-size: 0.85em;
      font-weight: 500;
    }
    .type-string { background: #e3f2fd; color: #1976d2; }
    .type-number { background: #e8f5e9; color: #388e3c; }
    .type-boolean { background: #f3e5f5; color: #7b1fa2; }
    .type-array { background: #fff3e0; color: #f57c00; }
    .type-object { background: #ffebee; color: #d32f2f; }
    .badge-required {
      color: #d32f2f;
      font-weight: 600;
      font-size: 0.85em;
    }
    .badge-optional {
      color: #666;
      font-size: 0.85em;
    }
    .text-muted {
      color: #999;
      font-style: italic;
    }
    pre {
      background: #2d2d2d;
      color: #f8f8f2;
      padding: 20px;
      border-radius: 6px;
      overflow-x: auto;
      font-size: 0.9em;
      line-height: 1.5;
    }
    code {
      font-family: 'Monaco', 'Menlo', monospace;
    }
    @media print {
      body { display: block; }
      .sidebar { display: none; }
      .main-content { margin-left: 0; max-width: 100%%; padding: 20px; }
      .container { box-shadow: none; padding: 20px; }
      .api-section { page-break-inside: avoid; }
    }
  </style>
</head>
<body>
  <div class="sidebar">
    <h1>%s</h1>
    <div class="sidebar-meta">%s: %s</div>
    <div class="sidebar-section">
      <h3>%s</h3>
      %s
    </div>
  </div>
  <div class="main-content">
    <div class="container">
      %s
    </div>
  </div>
  <script>
    document.querySelectorAll('.sidebar-api-item').forEach(link => {
      link.addEventListener('click', function(e) {
        e.preventDefault();
        const targetId = this.getAttribute('href');
        const targetElement = document.querySelector(targetId);
        if (targetElement) {
          targetElement.scrollIntoView({ behavior: 'smooth', block: 'start' });
          document.querySelectorAll('.sidebar-api-item').forEach(item => item.classList.remove('active'));
          this.classList.add('active');
        }
      });
    });

    const observer = new IntersectionObserver((entries) => {
      entries.forEach(entry => {
        if (entry.isIntersecting) {
          const id = entry.target.getAttribute('id');
          document.querySelectorAll('.sidebar-api-item').forEach(item => {
            item.classList.remove('active');
            if (item.getAttribute('href') === '#' + id) {
              item.classList.add('active');
            }
          });
        }
      });
    }, { rootMargin: '-20%% 0px -70%% 0px' });

    document.querySelectorAll('.api-section').forEach(section => {
      observer.observe(section);
    });
  </script>
</body>
</html>`, locale, title, title, generatedAt, time.Now().Format(time.RFC3339), tableOfContents, sidebarItems.String(), apisHTML.String())
}
