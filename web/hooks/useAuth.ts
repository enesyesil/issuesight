import useSWR from 'swr';
import { api } from '@/lib/api';
import { User } from '@/lib/types';

export function useAuth() {
    const { data: user, error, mutate, isLoading } = useSWR<User>('/api/auth/me', () => api.auth.me(), {
        shouldRetryOnError: false,
        revalidateOnFocus: false,
    });

    const logout = async () => {
        await api.auth.logout();
        mutate(undefined, false); // Clear user data
    };

    return {
        user,
        isLoading,
        isError: error,
        isAuthenticated: !!user,
        logout,
        mutate,
    };
}
