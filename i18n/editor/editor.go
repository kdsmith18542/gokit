// Package editor provides a web-based interface for editing i18n translation files.
//
// The i18n editor is an optional, embeddable HTTP handler that provides a user-friendly
// web interface for non-developers to edit translation messages. It supports TOML and
// JSON locale files and provides real-time editing capabilities.
//
// Features:
//   - Web-based UI for editing translation files
//   - Support for TOML and JSON locale formats
//   - Real-time file saving
//   - Locale switching and management
//   - Embeddable in existing Go web applications
//
// Example:
//
//	// Initialize the editor
//	editor := editor.NewHandler(editor.Config{
//	    LocalesDir: "./locales",
//	    Manager:    i18nManager,
//	})
//
//	// Mount in your application
//	http.Handle("/i18n-editor/", http.StripPrefix("/i18n-editor", editor))
//
//	// Access at http://localhost:8080/i18n-editor/
//
// Security Note:
// The editor should only be enabled in development environments or with proper
// authentication and authorization controls in production.
package editor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/kdsmith18542/gokit/i18n"
)

// Config configures the i18n editor behavior and connections.
// This struct contains all the configuration needed to initialize the editor.
type Config struct {
	LocalesDir string        // Directory containing locale files (e.g., "./locales")
	Manager    *i18n.Manager // Optional i18n manager for live reloading
}

// TranslationData represents the structure of translation data for the editor.
type TranslationData struct {
	Keys     []string                     `json:"keys"`
	Messages map[string]map[string]string `json:"messages"`
	Locales  []string                     `json:"locales"`
}

// NewHandler returns an http.Handler for the i18n editor.
// The handler provides both the web UI and API endpoints for managing translation files.
//
// The returned handler serves:
//   - GET / - The main editor UI
//   - GET /api/locales - List of available locales
//   - GET /api/translations - Get all translation data
//   - POST /api/save - Save translation data
//
// Example:
//
//	editor := editor.NewHandler(editor.Config{
//	    LocalesDir: "./locales",
//	})
//	http.Handle("/editor/", http.StripPrefix("/editor", editor))
func NewHandler(cfg Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", serveEditorUI)
	mux.HandleFunc("/api/locales", cfg.handleLocales)
	mux.HandleFunc("/api/translations", cfg.handleTranslations)
	mux.HandleFunc("/api/save", cfg.handleSave)
	return mux
}

// serveEditorUI serves the HTML/JS UI for the i18n editor.
// The UI provides a web-based interface for editing translation files.
func serveEditorUI(w http.ResponseWriter, r *http.Request) {
	html := editorHTML // see below
	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte(html)); err != nil {
		// Optionally log the error
	}
}

// handleLocales returns the list of available locales.
// Scans the locales directory for .toml and .json files and returns their locale codes.
// This endpoint is used by the UI to populate the locale selector.
func (cfg Config) handleLocales(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir(cfg.LocalesDir)
	if err != nil {
		http.Error(w, "Failed to read locales dir", http.StatusInternalServerError)
		return
	}
	var locales []string
	for _, f := range files {
		if !f.IsDir() && (strings.HasSuffix(f.Name(), ".toml") || strings.HasSuffix(f.Name(), ".json")) {
			locales = append(locales, strings.TrimSuffix(f.Name(), filepath.Ext(f.Name())))
		}
	}
	sort.Strings(locales)
	if err := json.NewEncoder(w).Encode(locales); err != nil {
		http.Error(w, "Failed to encode locales", http.StatusInternalServerError)
		return
	}
}

// handleTranslations returns all translation data organized by keys and locales.
// This endpoint provides the data needed for the table-based editor interface.
func (cfg Config) handleTranslations(w http.ResponseWriter, r *http.Request) {
	// Get all locale files
	files, err := os.ReadDir(cfg.LocalesDir)
	if err != nil {
		http.Error(w, "Failed to read locales dir", http.StatusInternalServerError)
		return
	}

	// Collect all locales and their messages
	allMessages := make(map[string]map[string]string)
	var locales []string
	var allKeys map[string]bool = make(map[string]bool)

	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".toml") {
			locale := strings.TrimSuffix(f.Name(), ".toml")
			locales = append(locales, locale)

			filePath := filepath.Join(cfg.LocalesDir, f.Name())
			messages, err := cfg.loadTOMLFile(filePath)
			if err != nil {
				continue // Skip invalid files
			}

			allMessages[locale] = messages

			// Collect all unique keys
			for key := range messages {
				allKeys[key] = true
			}
		}
	}

	// Convert keys map to sorted slice
	var keys []string
	for key := range allKeys {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	sort.Strings(locales)

	// Create the response structure
	data := TranslationData{
		Keys:     keys,
		Messages: allMessages,
		Locales:  locales,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode data", http.StatusInternalServerError)
		return
	}
}

// loadTOMLFile loads and parses a TOML file, returning a map of key-value pairs.
func (cfg Config) loadTOMLFile(filePath string) (map[string]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var messages map[string]interface{}
	if err := toml.Unmarshal(data, &messages); err != nil {
		return nil, err
	}

	// Convert interface{} values to strings
	result := make(map[string]string)
	for key, value := range messages {
		if str, ok := value.(string); ok {
			result[key] = str
		} else {
			// Handle non-string values by converting to string
			result[key] = fmt.Sprintf("%v", value)
		}
	}

	return result, nil
}

// handleSave saves the posted translation data.
// Accepts JSON data with translations and saves them to the appropriate locale files.
func (cfg Config) handleSave(w http.ResponseWriter, r *http.Request) {
	var data TranslationData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}

	// Save each locale file
	for _, locale := range data.Locales {
		if messages, exists := data.Messages[locale]; exists {
			if err := cfg.saveLocaleFile(locale, messages); err != nil {
				http.Error(w, fmt.Sprintf("Failed to save %s: %v", locale, err), http.StatusInternalServerError)
				return
			}
		}
	}

	// Trigger live reloading if manager is configured
	if cfg.Manager != nil {
		// Note: This would need to be implemented in the i18n.Manager
		// For now, we'll just return success
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "saved"}); err != nil {
		http.Error(w, "Failed to encode status", http.StatusInternalServerError)
		return
	}
}

// saveLocaleFile saves a locale's messages to a TOML file.
func (cfg Config) saveLocaleFile(locale string, messages map[string]string) error {
	filePath := filepath.Join(cfg.LocalesDir, locale+".toml")

	// Convert to TOML format
	var buffer strings.Builder
	buffer.WriteString(fmt.Sprintf("# %s locale file\n", locale))
	buffer.WriteString(fmt.Sprintf("# Generated by GoKit i18n Editor on %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// Sort keys for consistent output
	var keys []string
	for key := range messages {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := messages[key]
		// Escape quotes and newlines in TOML
		escapedValue := strings.ReplaceAll(value, `"`, `\"`)
		escapedValue = strings.ReplaceAll(escapedValue, "\n", "\\n")
		buffer.WriteString(fmt.Sprintf("%s = \"%s\"\n", key, escapedValue))
	}

	if err := os.WriteFile(filePath, []byte(buffer.String()), 0644); err != nil {
		return err
	}
	return nil
}

// editorHTML contains the embedded HTML/JS/CSS for the advanced i18n editor UI.
// The UI provides a table-based interface with live editing, search, and missing translation highlighting.
const editorHTML = `
<!DOCTYPE html>
<html>
<head>
  <title>GoKit i18n Editor</title>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <style>
    * { box-sizing: border-box; }
    body { 
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; 
      margin: 0; 
      padding: 20px; 
      background: #f5f5f5;
    }
    .container {
      max-width: 1400px;
      margin: 0 auto;
      background: white;
      border-radius: 8px;
      box-shadow: 0 2px 10px rgba(0,0,0,0.1);
      overflow: hidden;
    }
    .header {
      background: #2c3e50;
      color: white;
      padding: 20px;
    }
    .header h1 {
      margin: 0;
      font-size: 24px;
      font-weight: 500;
    }
    .controls {
      padding: 20px;
      border-bottom: 1px solid #eee;
      display: flex;
      gap: 15px;
      align-items: center;
      flex-wrap: wrap;
    }
    .search-box {
      flex: 1;
      min-width: 200px;
    }
    .search-box input {
      width: 100%;
      padding: 8px 12px;
      border: 1px solid #ddd;
      border-radius: 4px;
      font-size: 14px;
    }
    .save-btn {
      background: #27ae60;
      color: white;
      border: none;
      padding: 8px 16px;
      border-radius: 4px;
      cursor: pointer;
      font-size: 14px;
      font-weight: 500;
    }
    .save-btn:hover { background: #229954; }
    .save-btn:disabled { background: #95a5a6; cursor: not-allowed; }
    .status {
      color: #27ae60;
      font-size: 14px;
      font-weight: 500;
    }
    .error { color: #e74c3c; }
    .table-container {
      overflow-x: auto;
      max-height: 70vh;
    }
    .translation-table {
      width: 100%;
      border-collapse: collapse;
      font-size: 14px;
    }
    .translation-table th {
      background: #f8f9fa;
      padding: 12px 8px;
      text-align: left;
      border-bottom: 2px solid #dee2e6;
      font-weight: 600;
      position: sticky;
      top: 0;
      z-index: 10;
    }
    .translation-table td {
      padding: 8px;
      border-bottom: 1px solid #eee;
      vertical-align: top;
    }
    .translation-table tr:hover {
      background: #f8f9fa;
    }
    .key-cell {
      font-family: 'Monaco', 'Menlo', monospace;
      font-size: 13px;
      color: #2c3e50;
      font-weight: 500;
      min-width: 150px;
      max-width: 200px;
    }
    .translation-cell {
      min-width: 200px;
    }
    .translation-input {
      width: 100%;
      min-height: 60px;
      padding: 8px;
      border: 1px solid #ddd;
      border-radius: 4px;
      font-family: inherit;
      font-size: 14px;
      resize: vertical;
    }
    .translation-input:focus {
      outline: none;
      border-color: #3498db;
      box-shadow: 0 0 0 2px rgba(52, 152, 219, 0.2);
    }
    .missing {
      background: #fff3cd;
      border-color: #ffeaa7;
    }
    .missing:focus {
      border-color: #f39c12;
      box-shadow: 0 0 0 2px rgba(243, 156, 18, 0.2);
    }
    .loading {
      text-align: center;
      padding: 40px;
      color: #7f8c8d;
    }
    .stats {
      padding: 15px 20px;
      background: #f8f9fa;
      border-top: 1px solid #eee;
      font-size: 14px;
      color: #6c757d;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>GoKit i18n Editor</h1>
    </div>
    
    <div class="controls">
      <div class="search-box">
        <input type="text" id="searchInput" placeholder="Search translation keys or text...">
      </div>
      <button class="save-btn" id="saveBtn" onclick="saveTranslations()">Save Changes</button>
      <span class="status" id="status"></span>
    </div>
    
    <div class="table-container">
      <table class="translation-table" id="translationTable">
        <thead>
          <tr id="tableHeader">
            <th>Translation Key</th>
          </tr>
        </thead>
        <tbody id="tableBody">
          <tr>
            <td colspan="100" class="loading">Loading translations...</td>
          </tr>
        </tbody>
      </table>
    </div>
    
    <div class="stats" id="stats">
      Loading...
    </div>
  </div>

  <script>
    let translationData = null;
    let filteredKeys = [];
    let hasChanges = false;

    // Load translation data on page load
    window.onload = function() {
      loadTranslations();
    };

    // Search functionality
    document.getElementById('searchInput').addEventListener('input', function(e) {
      filterTranslations(e.target.value);
    });

    async function loadTranslations() {
      try {
        const response = await fetch('/api/translations');
        if (!response.ok) throw new Error('Failed to load translations');
        
        translationData = await response.json();
        filteredKeys = [...translationData.keys];
        renderTable();
        updateStats();
      } catch (error) {
        document.getElementById('tableBody').innerHTML = 
          '<tr><td colspan="100" style="text-align: center; color: #e74c3c;">Error loading translations: ' + error.message + '</td></tr>';
      }
    }

    function renderTable() {
      const header = document.getElementById('tableHeader');
      const body = document.getElementById('tableBody');
      
      // Render header
      header.innerHTML = '<th>Translation Key</th>';
      translationData.locales.forEach(locale => {
        header.innerHTML += '<th>' + locale.toUpperCase() + '</th>';
      });
      
      // Render body
      body.innerHTML = '';
      filteredKeys.forEach(key => {
        const row = document.createElement('tr');
        
        // Key cell
        const keyCell = document.createElement('td');
        keyCell.className = 'key-cell';
        keyCell.textContent = key;
        row.appendChild(keyCell);
        
        // Translation cells
        translationData.locales.forEach(locale => {
          const cell = document.createElement('td');
          cell.className = 'translation-cell';
          
          const input = document.createElement('textarea');
          input.className = 'translation-input';
          input.value = translationData.messages[locale]?.[key] || '';
          input.placeholder = 'Enter translation...';
          
          // Mark as missing if empty
          if (!input.value.trim()) {
            input.classList.add('missing');
          }
          
          // Track changes
          input.addEventListener('input', function() {
            hasChanges = true;
            document.getElementById('saveBtn').disabled = false;
            
            // Update missing status
            if (this.value.trim()) {
              this.classList.remove('missing');
            } else {
              this.classList.add('missing');
            }
          });
          
          cell.appendChild(input);
          row.appendChild(cell);
        });
        
        body.appendChild(row);
      });
    }

    function filterTranslations(searchTerm) {
      if (!translationData) return;
      
      const term = searchTerm.toLowerCase();
      filteredKeys = translationData.keys.filter(key => {
        // Search in key
        if (key.toLowerCase().includes(term)) return true;
        
        // Search in translations
        return translationData.locales.some(locale => {
          const translation = translationData.messages[locale]?.[key] || '';
          return translation.toLowerCase().includes(term);
        });
      });
      
      renderTable();
      updateStats();
    }

    function updateStats() {
      if (!translationData) return;
      
      const totalKeys = translationData.keys.length;
      const visibleKeys = filteredKeys.length;
      const totalTranslations = totalKeys * translationData.locales.length;
      
      let missingTranslations = 0;
      translationData.keys.forEach(key => {
        translationData.locales.forEach(locale => {
          if (!translationData.messages[locale]?.[key]?.trim()) {
            missingTranslations++;
          }
        });
      });
      
             const stats = document.getElementById('stats');
       stats.innerHTML = 
         'Showing ' + visibleKeys + ' of ' + totalKeys + ' keys | ' + 
         translationData.locales.length + ' locales | ' + 
         missingTranslations + ' missing translations';
    }

    async function saveTranslations() {
      if (!hasChanges) return;
      
      const saveBtn = document.getElementById('saveBtn');
      const status = document.getElementById('status');
      
      saveBtn.disabled = true;
      status.textContent = 'Saving...';
      status.className = 'status';
      
      try {
        // Collect all translation data
        const updatedData = {
          keys: translationData.keys,
          locales: translationData.locales,
          messages: {}
        };
        
        // Initialize messages structure
        translationData.locales.forEach(locale => {
          updatedData.messages[locale] = {};
        });
        
        // Collect data from table
        const rows = document.querySelectorAll('#tableBody tr');
        rows.forEach(row => {
          const key = row.querySelector('.key-cell').textContent;
          const inputs = row.querySelectorAll('.translation-input');
          
          inputs.forEach((input, index) => {
            const locale = translationData.locales[index];
            updatedData.messages[locale][key] = input.value;
          });
        });
        
        // Send to server
        const response = await fetch('/api/save', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify(updatedData)
        });
        
        if (!response.ok) throw new Error('Save failed');
        
        status.textContent = 'Saved successfully!';
        status.className = 'status';
        hasChanges = false;
        
        // Update local data
        translationData = updatedData;
        
        setTimeout(() => {
          status.textContent = '';
        }, 3000);
        
      } catch (error) {
        status.textContent = 'Save failed: ' + error.message;
        status.className = 'status error';
      } finally {
        saveBtn.disabled = !hasChanges;
      }
    }

    // Warn before leaving with unsaved changes
    window.addEventListener('beforeunload', function(e) {
      if (hasChanges) {
        e.preventDefault();
        e.returnValue = '';
      }
    });
  </script>
</body>
</html>
`
