// Test the new dark theme error component
const testErrorComponent = async () => {
  console.log('🎯 Testing Dark Theme Signup Page Error Handling...\n');

  try {
    // Test invalid email to trigger form validation
    const invalidEmailResponse = await fetch('http://localhost:3000/api/simple-signup', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ 
        email: 'invalid-email', // Invalid email format
        tier: 'FREE' 
      })
    });

    if (!invalidEmailResponse.ok) {
      const error = await invalidEmailResponse.json();
      console.log('✅ Error handling working:', error);
    }

    // Test missing required field
    const missingFieldResponse = await fetch('http://localhost:3000/api/simple-signup', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ 
        // Missing email field
        tier: 'FREE' 
      })
    });

    if (!missingFieldResponse.ok) {
      const error = await missingFieldResponse.json();
      console.log('✅ Missing field validation working:', error);
    }

    // Test valid request for comparison
    const validResponse = await fetch('http://localhost:3000/api/simple-signup', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ 
        email: 'dark-theme-success@example.com',
        tier: 'PRO' 
      })
    });

    if (validResponse.ok) {
      const result = await validResponse.json();
      console.log('✅ Success case working:', {
        tier: result.tier,
        keyLength: result.key.length,
        expiresAt: result.expiresAt
      });
    }

  } catch (error) {
    console.error('❌ Test failed:', error.message);
  }
};

testErrorComponent();
