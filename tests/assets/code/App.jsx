/**
 * Main application demonstrating Counter component usage.
 */

import React, { useState } from 'react';
import Counter from './Counter';

const App = () => {
    const [count, setCount] = useState(0);
    const [history, setHistory] = useState([]);

    const handleIncrement = () => {
        const newCount = count + 1;
        setCount(newCount);
        setHistory([...history, `Incremented to ${newCount}`]);
    };

    const handleDecrement = () => {
        const newCount = count - 1;
        setCount(newCount);
        setHistory([...history, `Decremented to ${newCount}`]);
    };

    const handleReset = () => {
        setCount(0);
        setHistory([]);
    };

    const appStyle = {
        fontFamily: 'Arial, sans-serif',
        padding: '20px',
        maxWidth: '600px',
        margin: '0 auto'
    };

    const resetButtonStyle = {
        display: 'block',
        margin: '20px auto',
        padding: '10px 30px',
        backgroundColor: '#6c757d',
        color: 'white',
        border: 'none',
        borderRadius: '4px',
        cursor: 'pointer',
        fontSize: '16px'
    };

    return (
        <div style={appStyle}>
            <h1>Counter Demo</h1>
            <Counter
                count={count}
                onIncrement={handleIncrement}
                onDecrement={handleDecrement}
            />
            <button style={resetButtonStyle} onClick={handleReset}>
                Reset Counter
            </button>
            <div style={{ marginTop: '20px' }}>
                <h3>History:</h3>
                <ul>
                    {history.map((item, index) => (
                        <li key={index}>{item}</li>
                    ))}
                </ul>
            </div>
        </div>
    );
};

export default App;
