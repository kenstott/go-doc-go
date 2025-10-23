/**
 * Button component with customizable styles and click handling.
 */

import React from 'react';

interface ButtonProps {
    label: string;
    onClick: () => void;
    variant?: 'primary' | 'secondary' | 'danger';
    disabled?: boolean;
}

export const Button: React.FC<ButtonProps> = ({
    label,
    onClick,
    variant = 'primary',
    disabled = false
}) => {
    const getButtonStyle = (): React.CSSProperties => {
        const baseStyle: React.CSSProperties = {
            padding: '10px 20px',
            border: 'none',
            borderRadius: '4px',
            cursor: disabled ? 'not-allowed' : 'pointer',
            fontSize: '16px',
            fontWeight: 'bold',
            opacity: disabled ? 0.6 : 1,
        };

        const variants = {
            primary: { backgroundColor: '#007bff', color: 'white' },
            secondary: { backgroundColor: '#6c757d', color: 'white' },
            danger: { backgroundColor: '#dc3545', color: 'white' },
        };

        return { ...baseStyle, ...variants[variant] };
    };

    return (
        <button
            onClick={onClick}
            disabled={disabled}
            style={getButtonStyle()}
        >
            {label}
        </button>
    );
};
