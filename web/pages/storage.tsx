import { NextPage } from 'next';
import Link from 'next/link';
import StorageStats from '../components/StorageStats';
import { useState } from 'react';

const StorageDashboard: NextPage = () => {
  const [activeTab, setActiveTab] = useState<'stats' | 'upload' | 'manage'>('stats');

  return (
    <div style={{ 
      minHeight: '100vh', 
      background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
      padding: '20px'
    }}>
      <div style={{
        maxWidth: '1200px',
        margin: '0 auto',
        background: 'white',
        borderRadius: '16px',
        overflow: 'hidden',
        boxShadow: '0 20px 40px rgba(0,0,0,0.1)'
      }}>
        {/* Header */}
        <div style={{
          background: 'linear-gradient(135deg, #667eea, #764ba2)',
          color: 'white',
          padding: '32px',
          textAlign: 'center'
        }}>
          <h1 style={{ 
            fontSize: '2.5rem', 
            margin: '0 0 16px 0',
            fontWeight: 'bold'
          }}>
            🚀 Bitcoin Sprint Storage
          </h1>
          <p style={{ fontSize: '1.2rem', opacity: 0.9, margin: 0 }}>
            Enterprise-grade decentralized storage dashboard
          </p>
        </div>

        {/* Navigation Tabs */}
        <div style={{
          display: 'flex',
          borderBottom: '1px solid #e5e7eb',
          background: '#f8fafc'
        }}>
          <button
            onClick={() => setActiveTab('stats')}
            style={{
              flex: 1,
              padding: '16px',
              border: 'none',
              background: activeTab === 'stats' ? 'white' : 'transparent',
              color: activeTab === 'stats' ? '#3b82f6' : '#64748b',
              borderBottom: activeTab === 'stats' ? '3px solid #3b82f6' : '3px solid transparent',
              cursor: 'pointer',
              fontSize: '1rem',
              fontWeight: 'medium',
              transition: 'all 0.2s'
            }}
          >
            📊 Statistics
          </button>
          <button
            onClick={() => setActiveTab('upload')}
            style={{
              flex: 1,
              padding: '16px',
              border: 'none',
              background: activeTab === 'upload' ? 'white' : 'transparent',
              color: activeTab === 'upload' ? '#3b82f6' : '#64748b',
              borderBottom: activeTab === 'upload' ? '3px solid #3b82f6' : '3px solid transparent',
              cursor: 'pointer',
              fontSize: '1rem',
              fontWeight: 'medium',
              transition: 'all 0.2s'
            }}
          >
            📤 Upload
          </button>
          <button
            onClick={() => setActiveTab('manage')}
            style={{
              flex: 1,
              padding: '16px',
              border: 'none',
              background: activeTab === 'manage' ? 'white' : 'transparent',
              color: activeTab === 'manage' ? '#3b82f6' : '#64748b',
              borderBottom: activeTab === 'manage' ? '3px solid #3b82f6' : '3px solid transparent',
              cursor: 'pointer',
              fontSize: '1rem',
              fontWeight: 'medium',
              transition: 'all 0.2s'
            }}
          >
            🔧 Manage
          </button>
        </div>

        {/* Content */}
        <div style={{ minHeight: '400px' }}>
          {activeTab === 'stats' && <StorageStats />}
          
          {activeTab === 'upload' && (
            <div style={{ padding: '40px', textAlign: 'center' }}>
              <div style={{ marginBottom: '32px' }}>
                <h2 style={{ marginBottom: '16px', color: '#1f2937' }}>📤 Upload Files</h2>
                <p style={{ color: '#64748b', marginBottom: '32px' }}>
                  Upload files to decentralized storage networks with enterprise security
                </p>
              </div>
              
              <Link href="/ipfs-upload">
                <a style={{
                  display: 'inline-block',
                  padding: '16px 32px',
                  background: 'linear-gradient(135deg, #667eea, #764ba2)',
                  color: 'white',
                  textDecoration: 'none',
                  borderRadius: '8px',
                  fontSize: '1.1rem',
                  fontWeight: 'medium',
                  transition: 'transform 0.2s',
                  boxShadow: '0 4px 12px rgba(102, 126, 234, 0.4)'
                }}
                onMouseOver={(e) => e.currentTarget.style.transform = 'translateY(-2px)'}
                onMouseOut={(e) => e.currentTarget.style.transform = 'translateY(0)'}
                >
                  🚀 Open Upload Interface
                </a>
              </Link>

              <div style={{ 
                marginTop: '40px',
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
                gap: '20px'
              }}>
                <div style={{
                  padding: '20px',
                  border: '1px solid #e5e7eb',
                  borderRadius: '8px',
                  textAlign: 'center'
                }}>
                  <div style={{ fontSize: '2rem', marginBottom: '12px' }}>🌐</div>
                  <h3 style={{ marginBottom: '8px', color: '#1f2937' }}>IPFS</h3>
                  <p style={{ fontSize: '0.9rem', color: '#64748b', margin: 0 }}>
                    Distributed peer-to-peer storage
                  </p>
                </div>
                
                <div style={{
                  padding: '20px',
                  border: '1px solid #e5e7eb',
                  borderRadius: '8px',
                  textAlign: 'center'
                }}>
                  <div style={{ fontSize: '2rem', marginBottom: '12px' }}>🏹</div>
                  <h3 style={{ marginBottom: '8px', color: '#1f2937' }}>Arweave</h3>
                  <p style={{ fontSize: '0.9rem', color: '#64748b', margin: 0 }}>
                    Permanent data storage
                  </p>
                </div>
                
                <div style={{
                  padding: '20px',
                  border: '1px solid #e5e7eb',
                  borderRadius: '8px',
                  textAlign: 'center'
                }}>
                  <div style={{ fontSize: '2rem', marginBottom: '12px' }}>💾</div>
                  <h3 style={{ marginBottom: '8px', color: '#1f2937' }}>Filecoin</h3>
                  <p style={{ fontSize: '0.9rem', color: '#64748b', margin: 0 }}>
                    Incentivized storage network
                  </p>
                </div>
              </div>
            </div>
          )}

          {activeTab === 'manage' && (
            <div style={{ padding: '40px' }}>
              <h2 style={{ marginBottom: '24px', color: '#1f2937' }}>🔧 Storage Management</h2>
              
              <div style={{ 
                display: 'grid', 
                gap: '20px',
                marginBottom: '32px'
              }}>
                <div style={{
                  padding: '24px',
                  border: '1px solid #e5e7eb',
                  borderRadius: '8px',
                  background: 'white'
                }}>
                  <h3 style={{ marginBottom: '16px', color: '#374151' }}>🔍 Verification API</h3>
                  <p style={{ marginBottom: '16px', color: '#64748b' }}>
                    Verify file integrity and availability across storage networks
                  </p>
                  <div style={{
                    background: '#f8fafc',
                    padding: '12px',
                    borderRadius: '4px',
                    border: '1px solid #e2e8f0'
                  }}>
                    <code style={{ fontSize: '0.9rem', color: '#1e40af' }}>
                      GET /api/storage/verify/[hash]
                    </code>
                  </div>
                </div>

                <div style={{
                  padding: '24px',
                  border: '1px solid #e5e7eb',
                  borderRadius: '8px',
                  background: 'white'
                }}>
                  <h3 style={{ marginBottom: '16px', color: '#374151' }}>📊 Analytics API</h3>
                  <p style={{ marginBottom: '16px', color: '#64748b' }}>
                    Get detailed storage statistics and usage metrics
                  </p>
                  <div style={{
                    background: '#f8fafc',
                    padding: '12px',
                    borderRadius: '4px',
                    border: '1px solid #e2e8f0'
                  }}>
                    <code style={{ fontSize: '0.9rem', color: '#1e40af' }}>
                      GET /api/storage/stats
                    </code>
                  </div>
                </div>

                <div style={{
                  padding: '24px',
                  border: '1px solid #e5e7eb',
                  borderRadius: '8px',
                  background: 'white'
                }}>
                  <h3 style={{ marginBottom: '16px', color: '#374151' }}>📤 Upload API</h3>
                  <p style={{ marginBottom: '16px', color: '#64748b' }}>
                    Upload files with enterprise security and validation
                  </p>
                  <div style={{
                    background: '#f8fafc',
                    padding: '12px',
                    borderRadius: '4px',
                    border: '1px solid #e2e8f0'
                  }}>
                    <code style={{ fontSize: '0.9rem', color: '#1e40af' }}>
                      POST /api/storage/upload
                    </code>
                  </div>
                </div>
              </div>

              <div style={{
                padding: '20px',
                background: '#eff6ff',
                border: '1px solid #bfdbfe',
                borderRadius: '8px'
              }}>
                <h4 style={{ marginBottom: '12px', color: '#1e40af' }}>🔐 Enterprise Features</h4>
                <ul style={{ margin: 0, paddingLeft: '20px', color: '#1e40af' }}>
                  <li>Hardware-backed security validation</li>
                  <li>Multi-protocol storage redundancy</li>
                  <li>Automated integrity checking</li>
                  <li>Enterprise-grade rate limiting</li>
                  <li>Real-time storage monitoring</li>
                </ul>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default StorageDashboard;
