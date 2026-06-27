import React, {useState, useCallback} from 'react';
import {Box, Text, useApp, useInput, useStdin} from 'ink';

type View = 'home' | 'about';

export default function App() {
	const {exit} = useApp();
	const {isRawModeSupported} = useStdin();
	const [view, setView] = useState<View>('home');
	const [count, setCount] = useState(0);
	const [inputValue, setInputValue] = useState('');

	const navigate = useCallback(
		(next: View) => {
			setView(next);
			setInputValue('');
		},
		[setView, setInputValue],
	);

	const handleInput = useCallback(
		(input: string, key: {escape: boolean}) => {
			setInputValue(input);

			if (input === 'q') {
				exit();
				return;
			}

			if (key.escape) {
				navigate('home');
				return;
			}

			if (view === 'home') {
				if (input === 'a') {
					setCount(c => c + 1);
				} else if (input === 'd') {
					setCount(c => c - 1);
				} else if (input === 'b') {
					navigate('about');
				}
			} else if (view === 'about') {
				if (input === 'h') {
					navigate('home');
				}
			}
		},
		[exit, navigate, view],
	);

	return (
		<Box flexDirection="column" padding={1}>
			<Box marginBottom={1}>
				<Text bold color="cyan">
					Ink App Shell
				</Text>
			</Box>

			{view === 'home' ? (
				<HomeView count={count} inputValue={inputValue} />
			) : (
				<AboutView />
			)}

			<Box marginTop={1}>
				{isRawModeSupported ? (
					<Text dimColor>
						{view === 'home'
							? '[a] increment  [d] decrement  [b] about  [q] quit'
							: '[h] home  [esc] back  [q] quit'}
					</Text>
				) : (
					<Text dimColor color="yellow">
						Interactive input requires a TTY.
					</Text>
				)}
			</Box>

			{isRawModeSupported ? <KeyboardHandler onInput={handleInput} /> : null}
		</Box>
	);
}

function KeyboardHandler({
	onInput,
}: {
	onInput: (input: string, key: {escape: boolean}) => void;
}) {
	useInput(onInput);
	return null;
}

function HomeView({
	count,
	inputValue,
}: {
	count: number;
	inputValue: string;
}) {
	return (
		<Box flexDirection="column">
			<Text>
				Current count: <Text color="green">{count}</Text>
			</Text>
			{inputValue ? <Text dimColor>Last key: {inputValue}</Text> : null}
		</Box>
	);
}

function AboutView() {
	return (
		<Box flexDirection="column">
			<Text>A terminal UI built with Ink + React.</Text>
			<Text dimColor>State management via React hooks.</Text>
		</Box>
	);
}
