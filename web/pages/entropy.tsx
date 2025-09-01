import { NextPage } from 'next';
import Head from 'next/head';
import EntropyGenerator from '../components/EntropyGenerator';

const EntropyPage: NextPage = () => {
	return (
		<>
			<Head>
				<title>Entropy Generator - Bitcoin Sprint</title>
				<meta name="description" content="Generate cryptographically secure random numbers using hardware entropy sources" />
				<meta property="og:title" content="Entropy Generator - Bitcoin Sprint" />
				<meta property="og:description" content="Generate cryptographically secure random numbers using hardware entropy sources" />
			</Head>

			<div className="min-h-screen bg-gradient-to-b from-gray-900 to-black">
				<EntropyGenerator />
			</div>
		</>
	);
};

export default EntropyPage;
