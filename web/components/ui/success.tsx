import React from 'react';

interface SuccessProps {
  title?: string;
  message?: string;
  children?: React.ReactNode;
  className?: string;
  onDismiss?: () => void;
}

export const Success: React.FC<SuccessProps> = ({ 
  title = "Success!",
  message,
  children,
  className = "",
  onDismiss 
}) => {
  return (
    <div className={`bg-green-50 border border-green-200 rounded-lg p-4 ${className}`}>
      <div className="flex items-start">
        <div className="flex-shrink-0">
          <svg 
            className="h-5 w-5 text-green-400" 
            viewBox="0 0 20 20" 
            fill="currentColor"
            aria-hidden="true"
          >
            <path 
              fillRule="evenodd" 
              d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.236 4.53L7.53 10.53a.75.75 0 00-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" 
              clipRule="evenodd" 
            />
          </svg>
        </div>
        <div className="ml-3 flex-1">
          <h3 className="text-sm font-medium text-green-800">
            {title}
          </h3>
          {message && (
            <div className="mt-1 text-sm text-green-700">
              {message}
            </div>
          )}
          {children && (
            <div className="mt-2">
              {children}
            </div>
          )}
          {onDismiss && (
            <div className="mt-3">
              <button
                type="button"
                onClick={onDismiss}
                className="bg-green-100 text-green-800 px-3 py-1 rounded text-sm font-medium hover:bg-green-200 transition-colors"
              >
                Dismiss
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default Success;
