import { useState } from "react";
import Icon from "./Icon";

interface Props {
	url: string;
}

const getFaviconUrlWithProvider = (url: string, provider: string) => {
	try {
		const domain = new URL(url).hostname;
		const faviconUrl = `${provider}/${domain}.ico`;
		return faviconUrl;
	} catch (error) {
		return "";
	}
};

const LinkFavicon = (props: Props) => {
	const { url } = props;
	const faviconProvider = "https://icons.duckduckgo.com/ip3";
	const [faviconUrl, setFaviconUrl] = useState<string>(
		getFaviconUrlWithProvider(url, faviconProvider),
	);

	const handleImgError = () => {
		setFaviconUrl("");
	};

	return faviconUrl ? (
		<img
			className="w-full h-auto rounded"
			src={faviconUrl}
			decoding="async"
			loading="lazy"
			onError={handleImgError}
			alt="favicon"
		/>
	) : (
		<Icon.CircleSlash
			className="w-full h-auto text-gray-400"
			strokeWidth={1.5}
		/>
	);
};

export default LinkFavicon;
