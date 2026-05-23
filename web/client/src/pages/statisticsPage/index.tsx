import React, { useState, useEffect, useCallback } from 'react';
import { useAuth } from '../../contexts/AuthContextState';
import { apiFetch } from '../../utils/api';
import type { Photo, PerformanceMetrics, MedicalInsights } from '../../types/photo';
import {
    BarChart,
    Bar,
    XAxis,
    YAxis,
    CartesianGrid,
    Tooltip,
    Legend,
    ResponsiveContainer,
    PieChart,
    Pie,
    Cell,
} from 'recharts';

const StatisticsPage: React.FC = () => {
    const { isAdmin } = useAuth();
    const [serverInstanceId, setServerInstanceId] = useState('unknown');
    const snoozeStorageKey = `medicina_muncii_expiry_alert_snooze_until_${serverInstanceId}`;
    const [loading, setLoading] = useState(true);
    const [downloading, setDownloading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [photos, setPhotos] = useState<Photo[]>([]);
    const [performance, setPerformance] = useState<PerformanceMetrics | null>(null);
    const [medicalInsights, setMedicalInsights] = useState<MedicalInsights | null>(null);
    const [showExpiryPopup, setShowExpiryPopup] = useState(false);

    const [controlChartType, setControlChartType] = useState<'bar' | 'pie'>('bar');
    const [avizChartType, setAvizChartType] = useState<'bar' | 'pie'>('pie');

    // Default date range: Last 30 days
    const today = new Date();
    const thirtyDaysAgo = new Date();
    thirtyDaysAgo.setDate(today.getDate() - 30);

    const [startDate, setStartDate] = useState(thirtyDaysAgo.toISOString().slice(0, 10));
    const [endDate, setEndDate] = useState(today.toISOString().slice(0, 10));

    const fetchData = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            // Calculate timestamps
            const startTimestamp = Math.floor(new Date(startDate).getTime() / 1000);
            const endTimestamp = Math.floor(new Date(endDate).getTime() / 1000) + 86399;

            const queryParams = new URLSearchParams();
            queryParams.append('start', startTimestamp.toString());
            queryParams.append('end', endTimestamp.toString());

            const response = await apiFetch(`/photos?${queryParams.toString()}`);
            const performanceResponse = await apiFetch(`/photos/performance?${queryParams.toString()}`);
            const medicalInsightsResponse = await apiFetch('/photos/medical-insights');
            const brokerInfoResponse = await apiFetch('/broker-info');

            if (!response.ok) {
                throw new Error('Failed to fetch data');
            }
            if (!performanceResponse.ok) {
                throw new Error('Failed to fetch performance data');
            }
            if (!medicalInsightsResponse.ok) {
                throw new Error('Failed to fetch medical insights data');
            }
            if (!brokerInfoResponse.ok) {
                throw new Error('Failed to fetch broker info');
            }

            const data = await response.json();
            const performanceData = await performanceResponse.json();
            const medicalInsightsData = await medicalInsightsResponse.json();
            const brokerInfoData = await brokerInfoResponse.json();
            setPhotos(Array.isArray(data) ? data : []);
            setPerformance(performanceData);
            setMedicalInsights(medicalInsightsData);
            setServerInstanceId(brokerInfoData.server_instance_id || 'unknown');
        } catch (err) {
            console.error('Error fetching stats data:', err);
            setError('Failed to load statistics data');
        } finally {
            setLoading(false);
        }
    }, [startDate, endDate]);

    useEffect(() => {
        fetchData();
    }, [fetchData]);

    useEffect(() => {
        if (!medicalInsights || (medicalInsights.expiring_next_month_people ?? 0) <= 0) {
            setShowExpiryPopup(false);
            return;
        }

        const snoozeUntil = Number(localStorage.getItem(snoozeStorageKey) ?? '0');
        setShowExpiryPopup(Number.isFinite(snoozeUntil) ? Date.now() >= snoozeUntil : true);
    }, [medicalInsights, snoozeStorageKey]);

    const remindMeLater = () => {
        const snoozeUntil = Date.now() + 24 * 60 * 60 * 1000;
        localStorage.setItem(snoozeStorageKey, String(snoozeUntil));
        setShowExpiryPopup(false);
    };

    const downloadAnonymizedReport = async () => {
        setDownloading(true);
        setError(null);
        try {
            const startTimestamp = Math.floor(new Date(startDate).getTime() / 1000);
            const endTimestamp = Math.floor(new Date(endDate).getTime() / 1000) + 86399;

            const queryParams = new URLSearchParams();
            queryParams.append('start', startTimestamp.toString());
            queryParams.append('end', endTimestamp.toString());

            const response = await apiFetch(`/photos/anonymized/export?${queryParams.toString()}`);
            if (!response.ok) {
                throw new Error('Failed to download anonymized report');
            }

            const blob = await response.blob();
            const objectUrl = URL.createObjectURL(blob);
            const contentDisposition = response.headers.get('Content-Disposition') ?? '';
            const filenameMatch = contentDisposition.match(/filename="?([^";]+)"?/i);
            const filename = filenameMatch?.[1] ?? 'anonymized_dataset.json';

            const link = document.createElement('a');
            link.href = objectUrl;
            link.download = filename;
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
            URL.revokeObjectURL(objectUrl);
        } catch (err) {
            console.error('Error downloading anonymized report:', err);
            setError('Failed to download anonymized report');
        } finally {
            setDownloading(false);
        }
    };

    // Process data for charts
    const getControlStats = () => {
        const stats = {
            'Angajare': 0,
            'Periodic': 0,
            'Adaptare': 0,
            'Reluare': 0,
            'Supraveghere': 0,
            'Alte': 0,
        };

        photos.forEach(photo => {
            if (photo.medical_data?.control_angajare?.value) stats['Angajare']++;
            if (photo.medical_data?.control_periodic?.value) stats['Periodic']++;
            if (photo.medical_data?.control_adaptare?.value) stats['Adaptare']++;
            if (photo.medical_data?.control_reluare?.value) stats['Reluare']++;
            if (photo.medical_data?.control_supraveghere?.value) stats['Supraveghere']++;
            if (photo.medical_data?.control_alte?.value) stats['Alte']++;
        });

        return Object.entries(stats).map(([name, value]) => ({ name, value }));
    };

    const getAvizStats = () => {
        const stats = {
            'APT': 0,
            'APT Conditionat': 0,
            'Inapt Temporar': 0,
            'Inapt': 0,
        };

        photos.forEach(photo => {
            if (photo.medical_data?.aviz_apt?.value) stats['APT']++;
            if (photo.medical_data?.aviz_apt_conditionat?.value) stats['APT Conditionat']++;
            if (photo.medical_data?.aviz_inapt_temporar?.value) stats['Inapt Temporar']++;
            if (photo.medical_data?.aviz_inapt?.value) stats['Inapt']++;
        });

        return Object.entries(stats).map(([name, value]) => ({ name, value }));
    };

    const controlData = getControlStats();
    const avizData = getAvizStats();
    const expiringNames = medicalInsights?.expiring_next_month_names ?? [];

    const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042', '#8884d8', '#82ca9d'];

    const renderChartToggle = (currentType: 'bar' | 'pie', setType: (t: 'bar' | 'pie') => void) => (
        <div className="flex bg-gray-100 p-1 rounded-md">
            <button
                onClick={() => setType('bar')}
                className={`px-3 py-1 text-sm rounded-sm transition-colors ${currentType === 'bar' ? 'bg-white shadow-sm text-sky-600 font-medium' : 'text-gray-500 hover:text-gray-700'
                    }`}
            >
                Bar
            </button>
            <button
                onClick={() => setType('pie')}
                className={`px-3 py-1 text-sm rounded-sm transition-colors ${currentType === 'pie' ? 'bg-white shadow-sm text-sky-600 font-medium' : 'text-gray-500 hover:text-gray-700'
                    }`}
            >
                Pie
            </button>
        </div>
    );

    const renderBarChart = (data: { name: string, value: number }[]) => (
        <ResponsiveContainer width="100%" height="100%">
            <BarChart data={data}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="name" />
                <YAxis />
                <Tooltip />
                <Legend />
                <Bar dataKey="value" fill="#0ea5e9" name="Count" />
            </BarChart>
        </ResponsiveContainer>
    );

    const renderPieChart = (data: { name: string, value: number }[]) => (
        <ResponsiveContainer width="100%" height="100%">
            <PieChart>
                <Pie
                    data={data}
                    cx="50%"
                    cy="50%"
                    labelLine={false}
                    label={({ name, percent }) => `${name} ${(percent ? percent * 100 : 0).toFixed(0)}%`}
                    outerRadius={100}
                    fill="#8884d8"
                    dataKey="value"
                >
                    {data.map((_, index) => (
                        <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                </Pie>
                <Tooltip />
            </PieChart>
        </ResponsiveContainer>
    );

    return (
        <div className="container mx-auto pb-10">
            {showExpiryPopup && (medicalInsights?.expiring_next_month_people ?? 0) > 0 && (
                <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/65 p-4">
                    <div className="w-full max-w-xl rounded-xl border-2 border-red-500 bg-white shadow-2xl">
                        <div className="rounded-t-lg bg-red-600 px-5 py-4 text-white">
                            <div className="text-lg font-bold uppercase tracking-wide">Urgent Expiration Alert</div>
                            <div className="mt-1 text-sm text-red-100">
                                {medicalInsights?.expiring_next_month_people ?? 0} people have medicina muncii expiring next month.
                            </div>
                        </div>
                        <div className="px-5 py-4">
                            <div className="text-sm font-semibold text-gray-800 mb-2">People affected:</div>
                            <div className="max-h-48 overflow-y-auto rounded border border-red-100 bg-red-50 px-3 py-2">
                                <ul className="list-disc pl-5 text-sm text-red-900 space-y-1">
                                    {(medicalInsights?.expiring_next_month_names ?? []).map((name) => (
                                        <li key={name}>{name}</li>
                                    ))}
                                </ul>
                            </div>
                            <div className="mt-4 flex justify-end">
                                <button
                                    onClick={remindMeLater}
                                    className="rounded-md border border-red-300 bg-white px-4 py-2 text-sm font-semibold text-red-700 hover:bg-red-100"
                                >
                                    Remind me later (24h)
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            )}

            <h1 className="text-2xl font-semibold text-sky-700 mb-6">Statistics</h1>

            {/* Date Filter */}
            <div className="bg-white p-4 rounded-lg shadow-sm mb-6 flex gap-4 items-end">
                <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Start Date</label>
                    <input
                        type="date"
                        value={startDate}
                        onChange={(e) => setStartDate(e.target.value)}
                        className="px-3 py-2 border border-gray-300 rounded-md focus:ring-sky-500"
                    />
                </div>
                <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">End Date</label>
                    <input
                        type="date"
                        value={endDate}
                        onChange={(e) => setEndDate(e.target.value)}
                        className="px-3 py-2 border border-gray-300 rounded-md focus:ring-sky-500"
                    />
                </div>
                <button
                    onClick={fetchData}
                    className="px-4 py-2 bg-sky-600 text-white rounded-md hover:bg-sky-700 mb-[1px]"
                >
                    Refresh
                </button>
                {isAdmin && (
                    <button
                        onClick={downloadAnonymizedReport}
                        disabled={downloading}
                        className="px-4 py-2 bg-emerald-600 text-white rounded-md hover:bg-emerald-700 disabled:opacity-60 disabled:cursor-not-allowed mb-[1px]"
                    >
                        {downloading ? 'Downloading...' : 'Download Anonymized Report'}
                    </button>
                )}
            </div>

            {loading ? (
                <div className="flex justify-center h-40 items-center">
                    <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-sky-500"></div>
                </div>
            ) : error ? (
                <div className="bg-red-50 text-red-700 p-4 rounded-md">{error}</div>
            ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-8">

                    {/* Control Type Chart */}
                    <div className="bg-white p-6 rounded-lg shadow-md">
                        <div className="flex justify-between items-center mb-4">
                            <h3 className="text-lg font-medium text-gray-800">Control Type Distribution</h3>
                            {renderChartToggle(controlChartType, setControlChartType)}
                        </div>
                        <div className="h-[300px]">
                            {controlChartType === 'bar' ? renderBarChart(controlData) : renderPieChart(controlData)}
                        </div>
                    </div>

                    {/* Aviz Medical Chart */}
                    <div className="bg-white p-6 rounded-lg shadow-md">
                        <div className="flex justify-between items-center mb-4">
                            <h3 className="text-lg font-medium text-gray-800">Medical Opinion Results</h3>
                            {renderChartToggle(avizChartType, setAvizChartType)}
                        </div>
                        <div className="h-[300px]">
                            {avizChartType === 'bar' ? renderBarChart(avizData) : renderPieChart(avizData)}
                        </div>
                    </div>

                    {/* Summary Cards */}
                    <div className="col-span-1 md:col-span-2 grid grid-cols-1 sm:grid-cols-3 gap-4">
                        <div className="bg-blue-50 p-4 rounded-lg border border-blue-100">
                            <span className="block text-sm text-blue-600 font-medium">Total Files</span>
                            <span className="block text-2xl font-bold text-blue-900">{photos.length}</span>
                        </div>
                        <div className="bg-green-50 p-4 rounded-lg border border-green-100">
                            <span className="block text-sm text-green-600 font-medium">FIT (APT)</span>
                            <span className="block text-2xl font-bold text-green-900">
                                {avizData.find(d => d.name === 'APT')?.value || 0}
                            </span>
                        </div>
                        <div className="bg-orange-50 p-4 rounded-lg border border-orange-100">
                            <span className="block text-sm text-orange-600 font-medium">Periodic Checks</span>
                            <span className="block text-2xl font-bold text-orange-900">
                                {controlData.find(d => d.name === 'Periodic')?.value || 0}
                            </span>
                        </div>
                    </div>

                    {/* Performance Metrics */}
                    <div className="col-span-1 md:col-span-2 bg-white p-6 rounded-lg shadow-md border border-slate-200">
                        <h3 className="text-lg font-medium text-gray-800 mb-4">System Performance</h3>
                        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                            <div className="bg-slate-50 p-4 rounded-lg border border-slate-200">
                                <span className="block text-sm text-slate-600 font-medium">OCR Success Rate</span>
                                <span className="block text-2xl font-bold text-slate-900">
                                    {(performance?.ocr_success_rate ?? 0).toFixed(2)}%
                                </span>
                                <span className="block text-xs text-slate-500 mt-1">
                                    {performance?.ocr_success_count ?? 0} / {performance?.total_documents ?? 0} successful
                                </span>
                            </div>
                            <div className="bg-indigo-50 p-4 rounded-lg border border-indigo-100">
                                <span className="block text-sm text-indigo-700 font-medium">Avg Processing Latency</span>
                                <span className="block text-2xl font-bold text-indigo-900">
                                    {(performance?.average_latency_ms ?? 0).toFixed(0)} ms
                                </span>
                            </div>
                            <div className="bg-cyan-50 p-4 rounded-lg border border-cyan-100">
                                <span className="block text-sm text-cyan-700 font-medium">P95 Processing Latency</span>
                                <span className="block text-2xl font-bold text-cyan-900">
                                    {performance?.p95_latency_ms ?? 0} ms
                                </span>
                            </div>
                        </div>
                    </div>

                    <div className="col-span-1 md:col-span-2 bg-white p-6 rounded-lg shadow-md border border-teal-200">
                        <h3 className="text-lg font-medium text-gray-800 mb-4">Medicina Muncii Insights</h3>
                        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                            <div className="bg-teal-50 p-4 rounded-lg border border-teal-100">
                                <span className="block text-sm text-teal-700 font-medium">Last Month</span>
                                <span className="block text-2xl font-bold text-teal-900">
                                    {medicalInsights?.last_month_count ?? 0}
                                </span>
                                <span className="block text-xs text-teal-600 mt-1">documents issued in the last 30 days</span>
                            </div>
                            <div
                                className="group relative bg-amber-50 p-4 rounded-lg border border-amber-100 cursor-help"
                                title={expiringNames.length > 0 ? `Expiring people: ${expiringNames.join(', ')}` : 'No upcoming expirations'}
                            >
                                <span className="block text-sm text-amber-700 font-medium">Expiring Next Month</span>
                                <span className="block text-2xl font-bold text-amber-900">
                                    {medicalInsights?.expiring_next_month_people ?? 0}
                                </span>
                                <span className="block text-xs text-amber-600 mt-1">unique people with expiry next calendar month</span>
                                {expiringNames.length > 0 && (
                                    <div className="pointer-events-none absolute bottom-full left-0 z-10 mb-2 hidden w-72 rounded-md border border-amber-300 bg-white p-3 text-xs text-gray-800 shadow-lg group-hover:block">
                                        <div className="mb-1 font-semibold text-amber-800">People expiring next month:</div>
                                        <ul className="list-disc pl-4 space-y-1">
                                            {expiringNames.map((name) => (
                                                <li key={name}>{name}</li>
                                            ))}
                                        </ul>
                                    </div>
                                )}
                            </div>
                            <div className="bg-sky-50 p-4 rounded-lg border border-sky-100">
                                <span className="block text-sm text-sky-700 font-medium">All Documents Total</span>
                                <span className="block text-2xl font-bold text-sky-900">
                                    {medicalInsights?.total_medicina_muncii_in_documents ?? 0}
                                </span>
                                <span className="block text-xs text-sky-600 mt-1">medicina muncii records across accessible documents</span>
                            </div>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};

export default StatisticsPage;
