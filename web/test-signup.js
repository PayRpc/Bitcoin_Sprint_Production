// Debug script to test API key generation
const testSignup = async () => {
  try {
    console.log('Testing signup API...');
    
    const response = await fetch('http://localhost:3000/api/signup', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        email: 'test@example.com',
        company: 'Test Company',
        tier: 'FREE'
      })
    });

    console.log('Response status:', response.status);
    console.log('Response headers:', Object.fromEntries(response.headers.entries()));

    if (!response.ok) {
      const errorText = await response.text();
      console.error('Error response:', errorText);
      return;
    }

    const data = await response.json();
    console.log('Success response:', data);
    console.log('Generated API key:', data.key);
    
  } catch (error) {
    console.error('Network error:', error);
  }
};

testSignup();
