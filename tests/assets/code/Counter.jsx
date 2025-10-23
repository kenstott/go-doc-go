/**
 * Counter component with increment and decrement functionality.
 */

import React from 'react';

const Counter = ({ count, onIncrement, onDecrement}) => {
    const containerStyle = {
        padding: '20px',
        border: '2px solid #007bff',
        borderRadius: '8px',
        textAlign: 'center',
        maxWidth: '300px',
        margin: '20px auto'
    };

    const buttonStyle = {
        padding: '10px 20px',
        margin: '0 5px',
        border: 'none',
        borderRadius: '4px',
        cursor: 'pointer',
        fontSize: '16px',
        fontWeight: 'bold'
    };

    const incrementStyle = {
        ...buttonStyle,
        backgroundColor: '#28a745',
        color: 'white'
    };

    const decrementStyle = {
        ...buttonStyle,
        backgroundColor: '#dc3545',
        color: 'white'
    };

    return (
        <div style={containerStyle}>
            <h2>Counter Value: {count}</h2>
            <div>
                <button style={incrementStyle} onClick={onIncrement}>
                    +1
                </button>
                <button style={decrementStyle} onClick={onDecrement}>
                    -1
                </button>
            </div>
        </div>
    );
};

export default Counter;
