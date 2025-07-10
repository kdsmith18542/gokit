package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/kdsmith18542/gokit/upload"
	"github.com/kdsmith18542/gokit/upload/storage"
)

func main() {
	// Create storage backend
	localStorage := storage.NewLocal("/tmp/uploads")

	// Create resumable upload processor
	options := upload.Options{
		MaxFileSize:      100 * 1024 * 1024, // 100MB
		AllowedMIMETypes: []string{"image/jpeg", "image/png", "application/pdf"},
		MaxFiles:         1,
	}

	processor := upload.NewResumableProcessor(localStorage, options)

	// Set up HTTP server
	http.HandleFunc("/upload", processor.HandleResumableUpload)
	http.HandleFunc("/", handleHome)

	fmt.Println("Resumable upload server starting on :8080")
	fmt.Println("Visit http://localhost:8080 to test the upload")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Resumable Upload Demo</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        .upload-container { border: 2px dashed #ccc; padding: 20px; text-align: center; margin: 20px 0; }
        .progress { width: 100%; height: 20px; background-color: #f0f0f0; border-radius: 10px; overflow: hidden; }
        .progress-bar { height: 100%; background-color: #4CAF50; width: 0%; transition: width 0.3s; }
        .status { margin: 10px 0; padding: 10px; border-radius: 5px; }
        .status.success { background-color: #d4edda; color: #155724; }
        .status.error { background-color: #f8d7da; color: #721c24; }
        .status.info { background-color: #d1ecf1; color: #0c5460; }
        button { padding: 10px 20px; margin: 5px; border: none; border-radius: 5px; cursor: pointer; }
        button.primary { background-color: #007bff; color: white; }
        button.danger { background-color: #dc3545; color: white; }
        button:disabled { opacity: 0.6; cursor: not-allowed; }
    </style>
</head>
<body>
    <h1>Resumable Upload Demo</h1>
    <p>This demo shows how to use the GoKit resumable upload functionality.</p>
    
    <div class="upload-container">
        <input type="file" id="fileInput" accept="image/*,.pdf" style="display: none;">
        <button class="primary" onclick="document.getElementById('fileInput').click()">Select File</button>
        <div id="fileInfo" style="margin: 10px 0;"></div>
        
        <div id="uploadControls" style="display: none;">
            <button class="primary" onclick="startUpload()">Start Upload</button>
            <button class="danger" onclick="abortUpload()" id="abortBtn" style="display: none;">Abort Upload</button>
        </div>
        
        <div class="progress" id="progressContainer" style="display: none;">
            <div class="progress-bar" id="progressBar"></div>
        </div>
        
        <div id="status"></div>
    </div>

    <script>
        let currentFile = null;
        let uploadSession = null;
        let chunkSize = 1024 * 1024; // 1MB chunks
        
        document.getElementById('fileInput').addEventListener('change', function(e) {
            const file = e.target.files[0];
            if (file) {
                currentFile = file;
                document.getElementById('fileInfo').innerHTML = 
                    '<strong>Selected:</strong> ' + file.name + ' (' + formatBytes(file.size) + ')';
                document.getElementById('uploadControls').style.display = 'block';
            }
        });
        
        function formatBytes(bytes) {
            if (bytes === 0) return '0 Bytes';
            const k = 1024;
            const sizes = ['Bytes', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }
        
        function showStatus(message, type = 'info') {
            const statusDiv = document.getElementById('status');
            statusDiv.innerHTML = '<div class="status ' + type + '">' + message + '</div>';
        }
        
        function updateProgress(percent) {
            document.getElementById('progressBar').style.width = percent + '%';
        }
        
        async function startUpload() {
            if (!currentFile) {
                showStatus('No file selected', 'error');
                return;
            }
            
            try {
                // Initiate upload
                showStatus('Initiating upload...', 'info');
                
                const initiateResponse = await fetch('/upload', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        file_name: currentFile.name,
                        total_size: currentFile.size,
                        mime_type: currentFile.type,
                        chunk_size: chunkSize
                    })
                });
                
                if (!initiateResponse.ok) {
                    throw new Error('Failed to initiate upload');
                }
                
                uploadSession = await initiateResponse.json();
                showStatus('Upload initiated. Starting chunk upload...', 'info');
                
                // Show progress and abort button
                document.getElementById('progressContainer').style.display = 'block';
                document.getElementById('abortBtn').style.display = 'inline-block';
                
                // Upload chunks
                const totalChunks = Math.ceil(currentFile.size / chunkSize);
                let uploadedChunks = 0;
                
                for (let i = 0; i < totalChunks; i++) {
                    const start = i * chunkSize;
                    const end = Math.min(start + chunkSize, currentFile.size);
                    const chunk = currentFile.slice(start, end);
                    
                    const chunkResponse = await fetch('/upload?file_id=' + uploadSession.file_id + '&chunk_number=' + i, {
                        method: 'PUT',
                        body: chunk
                    });
                    
                    if (!chunkResponse.ok) {
                        throw new Error('Failed to upload chunk ' + i);
                    }
                    
                    uploadedChunks++;
                    const progress = (uploadedChunks / totalChunks) * 100;
                    updateProgress(progress);
                    showStatus('Uploaded chunk ' + uploadedChunks + ' of ' + totalChunks, 'info');
                }
                
                // Complete upload
                showStatus('All chunks uploaded. Finalizing...', 'info');
                
                const completeResponse = await fetch('/upload?file_id=' + uploadSession.file_id, {
                    method: 'POST'
                });
                
                if (!completeResponse.ok) {
                    throw new Error('Failed to complete upload');
                }
                
                const result = await completeResponse.json();
                showStatus('Upload completed successfully! File URL: ' + result.url, 'success');
                
                // Hide controls
                document.getElementById('abortBtn').style.display = 'none';
                
            } catch (error) {
                showStatus('Upload failed: ' + error.message, 'error');
            }
        }
        
        async function abortUpload() {
            if (!uploadSession) return;
            
            try {
                await fetch('/upload?file_id=' + uploadSession.file_id, {
                    method: 'DELETE'
                });
                
                showStatus('Upload aborted', 'info');
                document.getElementById('abortBtn').style.display = 'none';
                document.getElementById('progressContainer').style.display = 'none';
                updateProgress(0);
                
            } catch (error) {
                showStatus('Failed to abort upload: ' + error.message, 'error');
            }
        }
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
