import React, { useState, useEffect } from 'react';

interface StorageStats {
  totalFiles: number;
  totalSize: number;
  protocols: {
    ipfs: number;
    arweave: number;
    filecoin: number;
    bitcoin: number;
  };
  recent: Array<{
    name: string;
    hash: string;
    size: number;
    protocol: string;
    uploadedAt: string;
  }>;
}

const StorageStats: React.FC = () => {
  const [stats, setStats] = useState<StorageStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchStats();
  }, []);

  const fetchStats = async () => {
    try {
      setLoading(true);
      const response = await fetch('/api/storage/stats');
      const result = await response.json();
      
      if (result.success) {
        setStats(result.data);
      } else {
        setError('Failed to load statistics');
      }
    } catch (err) {
      setError('Network error loading statistics');
    } finally {
      setLoading(false);
    }
  };

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatDate = (dateString: string): string => {
    const date = new Date(dateString);
    return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
  };

  const getProtocolColor = (protocol: string): string => {
    const colors = {
      ipfs: '#65a3ff',
      arweave: '#ff6b6b', 
      filecoin: '#4ecdc4',
      bitcoin: '#f7931a'
    };
    return colors[protocol as keyof typeof colors] || '#64748b';
  };

  const getProtocolIcon = (protocol: string): string => {
    const icons = {
      ipfs: '🌐',
      arweave: '🏹',
      filecoin: '💾',
      bitcoin: '₿'
    };
    return icons[protocol as keyof typeof icons] || '📁';
  };

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '40px' }}>
        <div style={{ fontSize: '2rem', marginBottom: '16px' }}>📊</div>
        <p>Loading storage statistics...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ textAlign: 'center', padding: '40px', color: '#ef4444' }}>
        <div style={{ fontSize: '2rem', marginBottom: '16px' }}>⚠️</div>
        <p>{error}</p>
        <button 
          onClick={fetchStats}
          style={{
            marginTop: '16px',
            padding: '8px 16px',
            background: '#3b82f6',
            color: 'white',
            border: 'none',
            borderRadius: '4px',
            cursor: 'pointer'
          }}
        >
          Retry
        </button>
      </div>
    );
  }

  if (!stats) return null;

  return (
    <div style={{ padding: '20px' }}>
      <h2 style={{ marginBottom: '24px', color: '#1f2937' }}>📊 Storage Statistics</h2>
      
      {/* Overview Cards */}
      <div style={{ 
        display: 'grid', 
        gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
        gap: '16px',
        marginBottom: '32px'
      }}>
        <div style={{
          background: 'linear-gradient(135deg, #667eea, #764ba2)',
          color: 'white',
          padding: '20px',
          borderRadius: '8px',
          textAlign: 'center'
        }}>
          <h3 style={{ margin: '0 0 8px 0', fontSize: '2rem' }}>{stats.totalFiles}</h3>
          <p style={{ margin: 0, opacity: 0.9 }}>Total Files</p>
        </div>
        
        <div style={{
          background: 'linear-gradient(135deg, #f093fb, #f5576c)',
          color: 'white',
          padding: '20px',
          borderRadius: '8px',
          textAlign: 'center'
        }}>
          <h3 style={{ margin: '0 0 8px 0', fontSize: '1.5rem' }}>{formatBytes(stats.totalSize)}</h3>
          <p style={{ margin: 0, opacity: 0.9 }}>Total Storage</p>
        </div>
      </div>

      {/* Protocol Distribution */}
      <div style={{ marginBottom: '32px' }}>
        <h3 style={{ marginBottom: '16px', color: '#374151' }}>🔗 Protocol Distribution</h3>
        <div style={{ 
          display: 'grid', 
          gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))',
          gap: '12px'
        }}>
          {Object.entries(stats.protocols).map(([protocol, count]) => (
            <div key={protocol} style={{
              padding: '16px',
              border: '1px solid #e5e7eb',
              borderRadius: '8px',
              textAlign: 'center',
              background: 'white'
            }}>
              <div style={{ 
                fontSize: '1.5rem', 
                marginBottom: '8px',
                color: getProtocolColor(protocol)
              }}>
                {getProtocolIcon(protocol)}
              </div>
              <h4 style={{ 
                margin: '0 0 4px 0', 
                fontSize: '1.25rem',
                color: getProtocolColor(protocol)
              }}>
                {count}
              </h4>
              <p style={{ 
                margin: 0, 
                textTransform: 'uppercase', 
                fontSize: '0.8rem',
                color: '#6b7280'
              }}>
                {protocol}
              </p>
            </div>
          ))}
        </div>
      </div>

      {/* Recent Files */}
      <div>
        <h3 style={{ marginBottom: '16px', color: '#374151' }}>📁 Recent Files</h3>
        <div style={{ 
          background: 'white', 
          border: '1px solid #e5e7eb', 
          borderRadius: '8px',
          overflow: 'hidden'
        }}>
          {stats.recent.map((file, index) => (
            <div key={index} style={{
              padding: '16px',
              borderBottom: index < stats.recent.length - 1 ? '1px solid #f3f4f6' : 'none',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between'
            }}>
              <div style={{ flex: 1 }}>
                <div style={{ 
                  display: 'flex', 
                  alignItems: 'center', 
                  marginBottom: '4px' 
                }}>
                  <span style={{ 
                    marginRight: '8px', 
                    color: getProtocolColor(file.protocol) 
                  }}>
                    {getProtocolIcon(file.protocol)}
                  </span>
                  <strong style={{ color: '#1f2937' }}>{file.name}</strong>
                </div>
                <div style={{ fontSize: '0.875rem', color: '#6b7280' }}>
                  {formatBytes(file.size)} • {formatDate(file.uploadedAt)}
                </div>
              </div>
              <div style={{ textAlign: 'right' }}>
                <div style={{ 
                  fontSize: '0.75rem', 
                  color: getProtocolColor(file.protocol),
                  textTransform: 'uppercase',
                  fontWeight: 'bold',
                  marginBottom: '4px'
                }}>
                  {file.protocol}
                </div>
                <code style={{ 
                  fontSize: '0.75rem', 
                  color: '#6b7280',
                  background: '#f9fafb',
                  padding: '2px 4px',
                  borderRadius: '3px'
                }}>
                  {file.hash.length > 16 ? file.hash.substring(0, 16) + '...' : file.hash}
                </code>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default StorageStats;
