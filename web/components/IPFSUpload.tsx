import React, { useState, useRef } from 'react';
import { ipfsUploadClient, UploadResponse, UploadProgress } from '../../lib/ipfsUploadClient';

const IPFSUploadComponent: React.FC = () => {
  const [uploadState, setUploadState] = useState<'idle' | 'uploading' | 'success' | 'error'>('idle');
  const [uploadProgress, setUploadProgress] = useState<UploadProgress | null>(null);
  const [uploadResult, setUploadResult] = useState<UploadResponse | null>(null);
  const [errorMessage, setErrorMessage] = useState<string>('');
  const [dragActive, setDragActive] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileSelect = (files: FileList | null) => {
    if (!files || files.length === 0) return;
    
    const file = files[0];
    uploadFile(file);
  };

  const uploadFile = async (file: File) => {
    try {
      setUploadState('uploading');
      setUploadProgress(null);
      setErrorMessage('');
      setUploadResult(null);

      // Validate file
      const validation = await ipfsUploadClient.validateFile(file);
      if (!validation.valid) {
        setErrorMessage(validation.error || 'File validation failed');
        setUploadState('error');
        return;
      }

      // Upload file with progress tracking
      const result = await ipfsUploadClient.uploadFile(file, (progress) => {
        setUploadProgress(progress);
      });

      setUploadResult(result);
      setUploadState('success');
      
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : 'Upload failed');
      setUploadState('error');
    }
  };

  const handleDrag = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true);
    } else if (e.type === 'dragleave') {
      setDragActive(false);
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);
    
    if (e.dataTransfer.files) {
      handleFileSelect(e.dataTransfer.files);
    }
  };

  const resetUpload = () => {
    setUploadState('idle');
    setUploadProgress(null);
    setUploadResult(null);
    setErrorMessage('');
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  return (
    <div className="ipfs-upload-container">
      <style jsx>{`
        .ipfs-upload-container {
          max-width: 600px;
          margin: 0 auto;
          padding: 20px;
        }

        .upload-header {
          text-align: center;
          margin-bottom: 30px;
        }

        .upload-header h2 {
          color: #2563eb;
          margin-bottom: 10px;
        }

        .upload-zone {
          border: 2px dashed #cbd5e1;
          border-radius: 12px;
          padding: 40px 20px;
          text-align: center;
          transition: all 0.3s ease;
          cursor: pointer;
          background: #f8fafc;
        }

        .upload-zone.active {
          border-color: #2563eb;
          background: #eff6ff;
        }

        .upload-zone.uploading {
          border-color: #10b981;
          background: #ecfdf5;
        }

        .upload-zone.success {
          border-color: #10b981;
          background: #ecfdf5;
        }

        .upload-zone.error {
          border-color: #ef4444;
          background: #fef2f2;
        }

        .upload-icon {
          font-size: 3rem;
          margin-bottom: 16px;
          display: block;
        }

        .upload-text {
          font-size: 1.1rem;
          color: #64748b;
          margin-bottom: 12px;
        }

        .upload-subtext {
          font-size: 0.9rem;
          color: #94a3b8;
        }

        .progress-container {
          margin: 20px 0;
        }

        .progress-bar {
          width: 100%;
          height: 8px;
          background: #e2e8f0;
          border-radius: 4px;
          overflow: hidden;
        }

        .progress-fill {
          height: 100%;
          background: linear-gradient(90deg, #2563eb, #3b82f6);
          transition: width 0.3s ease;
        }

        .progress-text {
          text-align: center;
          margin-top: 8px;
          font-size: 0.9rem;
          color: #64748b;
        }

        .result-container {
          background: #f8fafc;
          border: 1px solid #e2e8f0;
          border-radius: 8px;
          padding: 20px;
          margin-top: 20px;
        }

        .result-item {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 8px 0;
          border-bottom: 1px solid #e2e8f0;
        }

        .result-item:last-child {
          border-bottom: none;
        }

        .result-label {
          font-weight: 500;
          color: #374151;
        }

        .result-value {
          font-family: monospace;
          background: #e5e7eb;
          padding: 4px 8px;
          border-radius: 4px;
          font-size: 0.85rem;
          word-break: break-all;
          max-width: 300px;
        }

        .gateway-link {
          color: #2563eb;
          text-decoration: none;
          font-size: 0.9rem;
        }

        .gateway-link:hover {
          text-decoration: underline;
        }

        .error-message {
          background: #fef2f2;
          border: 1px solid #fecaca;
          color: #b91c1c;
          padding: 12px;
          border-radius: 6px;
          margin-top: 16px;
        }

        .button-group {
          display: flex;
          gap: 12px;
          justify-content: center;
          margin-top: 20px;
        }

        .btn {
          padding: 10px 20px;
          border: none;
          border-radius: 6px;
          font-weight: 500;
          cursor: pointer;
          transition: all 0.2s ease;
        }

        .btn-primary {
          background: #2563eb;
          color: white;
        }

        .btn-primary:hover {
          background: #1d4ed8;
        }

        .btn-secondary {
          background: #6b7280;
          color: white;
        }

        .btn-secondary:hover {
          background: #4b5563;
        }

        .hidden-input {
          display: none;
        }
      `}</style>

      <div className="upload-header">
        <h2>📁 IPFS File Upload</h2>
        <p>Upload files to the InterPlanetary File System with enterprise security</p>
      </div>

      <div
        className={`upload-zone ${dragActive ? 'active' : ''} ${uploadState}`}
        onDragEnter={handleDrag}
        onDragLeave={handleDrag}
        onDragOver={handleDrag}
        onDrop={handleDrop}
        onClick={() => uploadState === 'idle' && fileInputRef.current?.click()}
      >
        {uploadState === 'idle' && (
          <>
            <span className="upload-icon">☁️</span>
            <div className="upload-text">
              {dragActive ? 'Drop your file here' : 'Click to upload or drag and drop'}
            </div>
            <div className="upload-subtext">
              Supports: Images, PDFs, JSON, Text files (max 100MB)
            </div>
          </>
        )}

        {uploadState === 'uploading' && (
          <>
            <span className="upload-icon">⏳</span>
            <div className="upload-text">Uploading to IPFS...</div>
            {uploadProgress && (
              <div className="progress-container">
                <div className="progress-bar">
                  <div 
                    className="progress-fill" 
                    style={{ width: `${uploadProgress.percentage}%` }}
                  />
                </div>
                <div className="progress-text">
                  {uploadProgress.percentage}% ({ipfsUploadClient.formatFileSize(uploadProgress.loaded)} / {ipfsUploadClient.formatFileSize(uploadProgress.total)})
                </div>
              </div>
            )}
          </>
        )}

        {uploadState === 'success' && (
          <>
            <span className="upload-icon">✅</span>
            <div className="upload-text">Upload successful!</div>
          </>
        )}

        {uploadState === 'error' && (
          <>
            <span className="upload-icon">❌</span>
            <div className="upload-text">Upload failed</div>
          </>
        )}
      </div>

      <input
        ref={fileInputRef}
        type="file"
        className="hidden-input"
        onChange={(e) => handleFileSelect(e.target.files)}
        accept=".txt,.json,.jpg,.jpeg,.png,.gif,.pdf,.csv,.zip"
      />

      {uploadResult && uploadState === 'success' && (
        <div className="result-container">
          <h3 style={{ marginBottom: '16px', color: '#059669' }}>📊 Upload Results</h3>
          
          <div className="result-item">
            <span className="result-label">CID:</span>
            <span className="result-value">{uploadResult.cid}</span>
          </div>
          
          <div className="result-item">
            <span className="result-label">File Hash:</span>
            <span className="result-value">{uploadResult.hash}</span>
          </div>
          
          <div className="result-item">
            <span className="result-label">Size:</span>
            <span className="result-value">{ipfsUploadClient.formatFileSize(uploadResult.size || 0)}</span>
          </div>
          
          <div className="result-item">
            <span className="result-label">Filename:</span>
            <span className="result-value">{uploadResult.filename}</span>
          </div>

          {uploadResult.cid && (
            <div className="result-item">
              <span className="result-label">Gateway URL:</span>
              <a 
                href={ipfsUploadClient.getIPFSGatewayUrl(uploadResult.cid)}
                target="_blank"
                rel="noopener noreferrer"
                className="gateway-link"
              >
                View on IPFS Gateway →
              </a>
            </div>
          )}
        </div>
      )}

      {errorMessage && (
        <div className="error-message">
          <strong>Error:</strong> {errorMessage}
        </div>
      )}

      {uploadState !== 'idle' && (
        <div className="button-group">
          <button className="btn btn-secondary" onClick={resetUpload}>
            Upload Another File
          </button>
          {uploadResult?.cid && (
            <button 
              className="btn btn-primary"
              onClick={() => window.open(`/api/storage/verify?cid=${uploadResult.cid}`, '_blank')}
            >
              Verify Storage
            </button>
          )}
        </div>
      )}
    </div>
  );
};

export default IPFSUploadComponent;
