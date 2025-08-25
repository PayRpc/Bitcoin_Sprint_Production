import React from 'react'

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'default' | 'outline'
  size?: 'default' | 'lg'
  asChild?: boolean
  children: React.ReactNode
}

export function Button({ 
  variant = 'default', 
  size = 'default', 
  asChild = false,
  className = '', 
  children, 
  ...props 
}: ButtonProps) {
  const baseClasses = 'inline-flex items-center justify-center rounded-md font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:opacity-50 disabled:pointer-events-none ring-offset-background'
  
  const variantClasses = {
    default: 'bg-primary text-primary-foreground hover:bg-primary/90',
    outline: 'border border-input hover:bg-accent hover:text-accent-foreground'
  }
  
  const sizeClasses = {
    default: 'h-10 py-2 px-4',
    lg: 'h-11 px-8'
  }
  
  const classes = `${baseClasses} ${variantClasses[variant]} ${sizeClasses[size]} ${className}`
  
  // If asChild is requested, render the child element (e.g. an <a>) and merge classes/props
  if (asChild && React.isValidElement(children)) {
    // Strip `asChild` from props so it doesn't appear on the DOM element
    const { asChild: _ac, ...restProps } = props as any;
    const child = React.cloneElement(children as React.ReactElement, {
      className: [classes, (children as any).props?.className].filter(Boolean).join(' '),
      ...restProps,
    });
    return child;
  }

  // Ensure asChild is not forwarded to DOM
  const { asChild: _ac, ...buttonProps } = props as any;

  return (
    <button className={classes} {...buttonProps}>
      {children}
    </button>
  )
}
