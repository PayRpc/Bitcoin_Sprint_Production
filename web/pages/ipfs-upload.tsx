import { NextPage } from 'next';
import IPFSUploadComponent from '../components/IPFSUpload';

const IPFSUploadPage: NextPage = () => {
  return (
    <div style={{ 
      minHeight: '100vh', 
      background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
      padding: '40px 20px'
    }}>
      <div style={{
        maxWidth: '800px',
        margin: '0 auto',
        background: 'white',
        borderRadius: '16px',
        padding: '40px',
        boxShadow: '0 20px 40px rgba(0,0,0,0.1)'
      }}>
        <div style={{ textAlign: 'center', marginBottom: '40px' }}>
          <h1 style={{ 
            fontSize: '2.5rem', 
            background: 'linear-gradient(135deg, #667eea, #764ba2)',
            WebkitBackgroundClip: 'text',
            WebkitTextFillColor: 'transparent',
            marginBottom: '16px'
          }}>
            🚀 Bitcoin Sprint IPFS
          </h1>
          <p style={{ fontSize: '1.2rem', color: '#64748b' }}>
            Enterprise-grade decentralized file storage and verification
          </p>
        </div>

        <IPFSUploadComponent />

        <div style={{ 
          marginTop: '40px', 
          padding: '20px',
          background: '#f8fafc',
          borderRadius: '8px',
          border: '1px solid #e2e8f0'
        }}>
          <h3 style={{ marginBottom: '16px', color: '#374151' }}>📚 How it works:</h3>
          <ol style={{ color: '#64748b', lineHeight: '1.6' }}>
            <li>Select or drag a file to upload to IPFS</li>
            <li>File is validated and uploaded to the distributed network</li>
            <li>Receive a unique Content ID (CID) for your file</li>
            <li>Use the CID to access your file from any IPFS gateway</li>
            <li>Verify storage integrity using our validation API</li>
          </ol>
        </div>

        <div style={{ 
          marginTop: '20px', 
          padding: '16px',
          background: '#eff6ff',
          borderRadius: '8px',
          border: '1px solid #bfdbfe'
        }}>
          <p style={{ margin: 0, color: '#1e40af', fontSize: '0.9rem' }}>
            <strong>🔐 Enterprise Security:</strong> All uploads are validated, hashed, and pinned 
            with enterprise-grade security. Files are accessible globally via IPFS gateways.
          </p>
        </div>
      </div>
    </div>
  );
};

export default IPFSUploadPage;
