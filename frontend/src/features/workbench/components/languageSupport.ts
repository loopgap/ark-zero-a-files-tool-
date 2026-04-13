import type { Extension } from '@codemirror/state';
import { LanguageDescription } from '@codemirror/language';

let languageDescriptionsPromise: Promise<LanguageDescription[]> | null = null;

async function loadLanguageDescriptions() {
	if (!languageDescriptionsPromise) {
		languageDescriptionsPromise = import('@codemirror/language-data').then((module) => module.languages);
	}
	return languageDescriptionsPromise;
}

export async function loadLanguageExtension(path: string): Promise<Extension> {
	const descriptions = await loadLanguageDescriptions();
	const description = LanguageDescription.matchFilename(descriptions, path);
	if (!description) {
		return [];
	}
	try {
		return (await description.load()).extension;
	} catch {
		return [];
	}
}
