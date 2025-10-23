/**
 * Main application component demonstrating Button usage.
 */

import React, { useState } from 'react';
import { Button } from './Button';

const App: React.FC = () => {
    const [count, setCount] = useState<number>(0);
    const [message, setMessage] = useState<string>('');

    const handleIncrement = () => {
        setCount(count + 1);
        setMessage(`Count incremented to ${count + 1}`);
    };

    const handleDecrement = () => {
        setCount(count - 1);
        setMessage(`Count decremented to ${count - 1}`);
    };

    const handleReset = () => {
        setCount(0);
        setMessage('Count reset to 0');
    };

    return (
        <div style={{ padding: '20px', fontFamily: 'Arial, sans-serif' }}>
            <h1>Button Component Demo</h1>
            <div style={{ marginBottom: '20px' }}>
                <h2>Counter: {count}</h2>
                <p>{message}</p>
            </div>
            <div style={{ display: 'flex', gap: '10px' }}>
                <Button label="Increment" onClick={handleIncrement} variant="primary" />
                <Button label="Decrement" onClick={handleDecrement} variant="secondary" />
                <Button label="Reset" onClick={handleReset} variant="danger" />
            </div>
        </div>
    );
};

export default App;
