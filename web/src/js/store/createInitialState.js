import loadState from "../storage/loadState";

export default () => {

    const persistedState = loadState();

    return {
        loading: false,
        channels: {},
        apiBaseUrl: process.env.apiBaseUrl,
        ...persistedState,
    }
}