import { getMaintenanceStatus } from '@/lib/maintenance';
import { GetServerSideProps } from 'next';
import { useEffect, useState } from 'react';

interface MaintenancePageProps {
  maintenance: {
    enabled: boolean;
    reason?: string;
    started_at?: string;
    estimated_duration?: string;
  };
}

export default function MaintenancePage({ maintenance }: MaintenancePageProps) {
  const [timeElapsed, setTimeElapsed] = useState<string>('');

  useEffect(() => {
    if (!maintenance.started_at) return;

    const updateElapsed = () => {
      const start = new Date(maintenance.started_at!);
      const now = new Date();
      const elapsed = Math.floor((now.getTime() - start.getTime()) / 1000 / 60); // minutes
      setTimeElapsed(`${elapsed} minutes`);
    };

    updateElapsed();
    const interval = setInterval(updateElapsed, 60000); // Update every minute

    return () => clearInterval(interval);
  }, [maintenance.started_at]);

  return (
    <div className="min-h-screen bg-gradient-to-br from-orange-400 via-orange-500 to-yellow-600 flex items-center justify-center p-4">
      <div className="max-w-md w-full bg-white rounded-lg shadow-xl p-8 text-center">
        {/* Bitcoin Sprint Logo */}
        <div className="mb-6">
          <div className="w-16 h-16 bg-orange-500 rounded-full mx-auto flex items-center justify-center mb-4">
            <svg className="w-8 h-8 text-white" fill="currentColor" viewBox="0 0 24 24">
              <path d="M23.638 14.904c-1.602 6.43-8.113 10.34-14.542 8.736C2.67 22.05-1.244 15.525.362 9.105 1.962 2.67 8.475-1.243 14.9.358c6.43 1.605 10.342 8.115 8.738 14.546z"/>
              <path fill="#fff" d="M17.002 11.6c.168-1.13-.693-1.737-1.871-2.143l.382-1.532-0.934-.233-.372 1.493c-.245-.061-.497-.119-.747-.176l.375-1.504-0.934-.233-.382 1.532c-.203-.046-.403-.092-.596-.14l0-.008-1.287-.321-.248.996s.693.159.678.169c.378.094.446.344.435.543l-.435 1.743c.026.007.06.016.098.03l-.101-.025-.611 2.45c-.046.115-.164.287-.429.222.009.013-.678-.169-.678-.169l-.464 1.067 1.215.303c.226.056.447.115.665.171l-.387 1.554.934.233.382-1.532c.255.069.502.133.744.196l-.381 1.525.934.233.387-1.552c1.595.302 2.793.18 3.297-1.26.406-1.16-.02-1.829-.857-2.267.61-.141 1.07-.544 1.194-1.377zm-2.138 2.998c-.289 1.16-2.243.533-2.876.376l.513-2.057c.633.158 2.667.471 2.363 1.681zm.289-3.016c-.263 1.056-1.89.52-2.417.388l.465-1.865c.527.132 2.232.378 1.952 1.477z"/>
            </svg>
          </div>
          <h1 className="text-2xl font-bold text-gray-900">Bitcoin Sprint</h1>
        </div>

        {/* Maintenance Message */}
        <div className="mb-6">
          <div className="w-12 h-12 bg-orange-100 rounded-full mx-auto flex items-center justify-center mb-4">
            <svg className="w-6 h-6 text-orange-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
            </svg>
          </div>
          <h2 className="text-xl font-semibold text-gray-900 mb-2">System Maintenance</h2>
          <p className="text-gray-600 mb-4">
            {maintenance.reason || 'We are currently performing system maintenance to improve your experience.'}
          </p>
        </div>

        {/* Maintenance Details */}
        <div className="bg-gray-50 rounded-lg p-4 mb-6">
          <div className="space-y-2 text-sm">
            {maintenance.started_at && (
              <div className="flex justify-between">
                <span className="text-gray-500">Started:</span>
                <span className="text-gray-900">
                  {new Date(maintenance.started_at).toLocaleString()}
                </span>
              </div>
            )}
            {timeElapsed && (
              <div className="flex justify-between">
                <span className="text-gray-500">Time elapsed:</span>
                <span className="text-gray-900">{timeElapsed}</span>
              </div>
            )}
            {maintenance.estimated_duration && (
              <div className="flex justify-between">
                <span className="text-gray-500">Estimated duration:</span>
                <span className="text-gray-900">{maintenance.estimated_duration}</span>
              </div>
            )}
          </div>
        </div>

        {/* Status */}
        <div className="text-center">
          <p className="text-sm text-gray-500 mb-4">
            We apologize for any inconvenience. Please check back shortly.
          </p>
          <button
            onClick={() => window.location.reload()}
            className="bg-orange-500 hover:bg-orange-600 text-white font-medium py-2 px-4 rounded-lg transition-colors duration-200"
          >
            Check Again
          </button>
        </div>

        {/* Footer */}
        <div className="mt-8 pt-4 border-t border-gray-200">
          <p className="text-xs text-gray-400">
            For urgent support, please contact our technical team.
          </p>
        </div>
      </div>
    </div>
  );
}

export const getServerSideProps: GetServerSideProps = async () => {
  try {
    const maintenance = await getMaintenanceStatus();
    
    // If maintenance is not enabled, redirect to home
    if (!maintenance?.enabled) {
      return {
        redirect: {
          destination: '/',
          permanent: false,
        },
      };
    }

    return {
      props: {
        maintenance,
      },
    };
  } catch (error) {
    // If we can't check maintenance status, redirect to home
    return {
      redirect: {
        destination: '/',
        permanent: false,
      },
    };
  }
};
